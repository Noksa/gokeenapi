package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Local exporter infrastructure", func() {
	It("rejects missing config before starting Docker", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, "make", "exporter-local-up")
		command.Env = append(os.Environ(), "EXPORTER_CONFIG=")
		output, err := command.CombinedOutput()
		Expect(err).To(HaveOccurred())
		Expect(string(output)).To(ContainSubstring("EXPORTER_CONFIG"))
	})
	It("provisions Prometheus and dashboard, scrapes cached exporter metrics, and stops without a config", func() {
		if _, err := exec.LookPath("docker"); err != nil {
			Skip("Docker CLI unavailable")
		}
		cfgDir := createDockerAccessibleTempDir(GinkgoT())
		Expect(os.Chmod(cfgDir, 0o700)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(cfgDir, "router.yaml"), []byte("keenetic:\n  url: http://127.0.0.1:1\n  login: test-monitor\n  password: fixture-password\n"), 0o600)).To(Succeed())
		configPath := filepath.Join(cfgDir, "exporter.yaml")
		original := []byte("listen: 127.0.0.1:19479\nrouters:\n  - {name: fixture, config: router.yaml}\n")
		Expect(os.WriteFile(configPath, original, 0o600)).To(Succeed())
		project := fmt.Sprintf("gokeenapi-exporter-test-%d", os.Getpid())
		env := append(os.Environ(), "EXPORTER_CONFIG="+configPath, "EXPORTER_PROJECT="+project, "EXPORTER_PORT=0", "PROMETHEUS_PORT=0", "GRAFANA_PORT=0")
		run := func(action string) ([]byte, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "-c", `umask 077; exec make "exporter-local-$1"`, "exporter-local-test", action)
			cmd.Env = env
			return cmd.CombinedOutput()
		}
		DeferCleanup(func() {
			output, err := run("down")
			Expect(err).NotTo(HaveOccurred(), string(output))
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			for _, suffix := range []string{"prometheus-data", "grafana-data"} {
				_ = exec.CommandContext(ctx, "docker", "volume", "rm", project+"_"+suffix).Run()
			}
		})
		output, err := run("up")
		Expect(err).NotTo(HaveOccurred(), string(output))
		runtimeDir := filepath.Join(mustWorkingDirectory(), "docker/infra-exporter-local/.runtime", project, "dashboards")
		dashboardInfo, err := os.Stat(filepath.Join(runtimeDir, "keenetic.json"))
		Expect(err).NotTo(HaveOccurred())
		Expect(dashboardInfo.Mode().Perm()&0o444).To(Equal(os.FileMode(0o444)), "Grafana uses a different UID; generated public dashboard assets must remain readable with umask 077")
		configInfo, err := os.Stat(configPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(configInfo.Mode().Perm()).To(Equal(os.FileMode(0o600)))
		address := func(service, port string) string {
			cmd := exec.Command("docker", "compose", "-p", project, "-f", "docker/infra-exporter-local/compose.yaml", "port", service, port)
			cmd.Env = append(env, "EXPORTER_CONFIG_DIR="+cfgDir, "EXPORTER_CONFIG_NAME=exporter.yaml", "EXPORTER_RUNTIME_DIR="+filepath.Join(mustWorkingDirectory(), "docker/infra-exporter-local/.runtime", project))
			b, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(b))
			return "http://" + strings.TrimSpace(string(b))
		}
		client := &http.Client{Timeout: 3 * time.Second}
		get := func(url string, target any) {
			r, err := client.Get(url)
			Expect(err).NotTo(HaveOccurred())
			defer func() { Expect(r.Body.Close()).To(Succeed()) }()
			Expect(r.StatusCode).To(Equal(http.StatusOK))
			Expect(json.NewDecoder(r.Body).Decode(target)).To(Succeed())
		}
		promURL := address("prometheus", "9090")
		Eventually(func() string {
			r, err := client.Get(promURL + "/api/v1/query?query=up%7Bjob%3D%22gokeenapi%22%7D")
			if err != nil {
				return ""
			}
			defer func() { Expect(r.Body.Close()).To(Succeed()) }()
			var result struct {
				Data struct {
					Result []struct {
						Value []any `json:"value"`
					} `json:"result"`
				} `json:"data"`
			}
			if json.NewDecoder(r.Body).Decode(&result) != nil || len(result.Data.Result) == 0 {
				return ""
			}
			return fmt.Sprint(result.Data.Result[0].Value[1])
		}).WithTimeout(40 * time.Second).WithPolling(time.Second).Should(Equal("1"))
		grafanaURL := address("grafana", "3000")
		var dashboard struct {
			Dashboard struct {
				Panels []struct {
					Datasource struct {
						UID string `json:"uid"`
					} `json:"datasource"`
				} `json:"panels"`
			} `json:"dashboard"`
		}
		get(grafanaURL+"/api/dashboards/uid/gokeenapi-keenetic", &dashboard)
		Expect(dashboard.Dashboard.Panels).NotTo(BeEmpty())
		for _, panel := range dashboard.Dashboard.Panels {
			Expect(panel.Datasource.UID).To(Equal("gokeenapi-prometheus"))
		}
		var datasource struct {
			URL string `json:"url"`
		}
		get(grafanaURL+"/api/datasources/uid/gokeenapi-prometheus", &datasource)
		Expect(datasource.URL).To(Equal("http://prometheus:9090"))
		after, err := os.ReadFile(configPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(after).To(Equal(original))
		// Bind-mounted file contents do not affect Compose's configuration hash.
		// A second up must reload in-place edits even when the image is cached.
		Expect(os.WriteFile(configPath, []byte(strings.ReplaceAll(string(original), "name: fixture", "name: updated")), 0o600)).To(Succeed())
		output, err = run("up")
		Expect(err).NotTo(HaveOccurred(), string(output))
		exporterURL := address("exporter", "9479")
		response, err := client.Get(exporterURL + "/metrics")
		Expect(err).NotTo(HaveOccurred())
		body, err := io.ReadAll(response.Body)
		Expect(response.Body.Close()).To(Succeed())
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(ContainSubstring(`router="updated"`))
		Expect(string(body)).NotTo(ContainSubstring(`router="fixture"`))
		env = append(env, "EXPORTER_CONFIG=")
		output, err = run("down")
		Expect(err).NotTo(HaveOccurred(), string(output))
	})
})

func mustWorkingDirectory() string {
	dir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	return dir
}
