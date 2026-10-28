package cmd

import (
	"fmt"

	"github.com/noksa/gokeenapi/internal/exporter"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/spf13/cobra"
)

func newExporterCmd() *cobra.Command {
	return &cobra.Command{
		Use: CmdExporter, Aliases: AliasesExporter,
		Short: "Expose cached multi-router metrics for Prometheus",
		Long: `Run a read-only Prometheus exporter for one or more Keenetic routers.

Polls run on independent schedules. GET /metrics serves cached snapshots
without issuing router API requests. Exporter YAML references existing router
YAML files; their routes and domain-source lists are not loaded.

Example:
  gokeenapi exporter --config exporter.yaml

The exporter listens on 127.0.0.1:9479 by default and runs until stopped.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			cfg, err := config.LoadExporterConfig(path)
			if err != nil {
				return fmt.Errorf("load exporter config: %w", err)
			}
			service, err := exporter.New(cfg)
			if err != nil {
				return err
			}
			gokeenlog.Infof("Exporter: %d router(s), metrics listener %s", len(cfg.Routers), cfg.Listen)
			return service.Serve(cmd.Context())
		},
	}
}
