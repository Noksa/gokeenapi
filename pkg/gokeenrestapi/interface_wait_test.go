package gokeenrestapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WaitUntilInterfaceIsUpContext", func() {
	It("should return immediately when interface is already up", func() {
		// Default mock has Wireguard0 in connected/up state
		server := SetupMockRouterForTest()
		DeferCleanup(server.Close)
		DeferCleanup(CleanupTestConfig)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := time.Now()
		Expect(Interface.WaitUntilInterfaceIsUpContext(ctx, "Wireguard0")).To(Succeed())
		Expect(time.Since(start)).To(BeNumerically("<", 3*time.Second))
	})

	It("should return error when context is cancelled before interface comes up", func() {
		// Start with Wireguard0 in down state
		server := SetupMockRouterForTest()
		DeferCleanup(server.Close)
		DeferCleanup(CleanupTestConfig)

		// Bring the interface down
		body, _ := json.Marshal([]gokeenrestapimodels.ParseRequest{
			{Parse: "interface Wireguard0 down"},
		})
		resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		Expect(err).NotTo(HaveOccurred())
		_ = resp.Body.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = Interface.WaitUntilInterfaceIsUpContext(ctx, "Wireguard0")
		Expect(err).To(HaveOccurred())
		// context deadline or cancellation
		Expect(err.Error()).To(SatisfyAny(
			ContainSubstring("context"),
			ContainSubstring("deadline"),
			ContainSubstring("still not up"),
		))
	})

	It("should succeed once interface becomes up during polling", func() {
		var callCount atomic.Int32

		// Custom server: first N polls return down, then up
		const transitionAfter = 3

		mux := http.NewServeMux()
		mockRouter := NewMockRouter()

		// Bring Wireguard0 down in the mock
		mockRouter.mu.Lock()
		if iface, ok := mockRouter.interfaces["Wireguard0"]; ok {
			iface.Connected = StateDisconnected
			iface.Link = StateDown
			iface.State = StateDown
		}
		mockRouter.mu.Unlock()

		mux.HandleFunc("/auth", mockRouter.handleAuth)
		mux.HandleFunc("/rci/show/version", mockRouter.handleVersion)
		mux.HandleFunc("/rci/show/interface/", func(w http.ResponseWriter, r *http.Request) {
			count := callCount.Add(1)
			if count >= transitionAfter {
				// Bring the interface up in the mock state
				mockRouter.mu.Lock()
				if iface, ok := mockRouter.interfaces["Wireguard0"]; ok {
					iface.Connected = StateConnected
					iface.Link = StateUp
					iface.State = StateUp
				}
				mockRouter.mu.Unlock()
			}
			mockRouter.handleInterface(w, r)
		})
		mux.HandleFunc("/rci/", mockRouter.handleParse)
		mux.HandleFunc("/rci/show/interface", mockRouter.handleInterfaces)

		srv := httptest.NewServer(mux)
		DeferCleanup(srv.Close)

		SetupTestConfig(srv.URL)
		DeferCleanup(CleanupTestConfig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		Expect(Interface.WaitUntilInterfaceIsUpContext(ctx, "Wireguard0")).To(Succeed())
		Expect(callCount.Load()).To(BeNumerically(">=", int32(transitionAfter)))
	})

	It("WaitUntilInterfaceIsUp (non-context variant) should succeed for an already-up interface", func() {
		server := SetupMockRouterForTest()
		DeferCleanup(server.Close)
		DeferCleanup(CleanupTestConfig)

		Expect(Interface.WaitUntilInterfaceIsUp("Wireguard0")).To(Succeed())
	})
})
