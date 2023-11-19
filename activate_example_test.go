// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package launchd_test

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/tprasadtp/go-launchd"
)

// WaitGroup to wait on multiple listeners.
var wg sync.WaitGroup

func ExampleTCPListenersWithName() {
	// This example only works on macOS, But is shown on all platforms
	// for ease of use. This cannot be used for systemd socket activation.
	listeners, err := launchd.TCPListenersWithName("socket-name-as-in-plist")
	if err != nil {
		slog.Error("Error getting socket activated listeners", "err", err)
		// Handle error and close any active listeners.
		for _, item := range listeners {
			item.Close()
		}
		os.Exit(1)
	}

	// A simple HTTP handler. Replace this with your actual implementation.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello From Socket Activated Server\n"))
		slog.Info("Request received",
			"client", r.RemoteAddr,
			"method", r.Method,
			"url", r.URL)
	})

	// Make servers stoppable with ctrl+x or SIGTERM.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Because there may be multiple listeners, we need to have as many servers as listeners.
	servers := make([]*http.Server, 0, len(listeners))
	for range listeners {
		servers = append(servers, &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: time.Second * 30,
			BaseContext: func(_ net.Listener) context.Context {
				return ctx
			},
		})
	}

	for i := 0; i < len(listeners); i++ {
		// Run servers in background
		wg.Add(1)
		go func(s *http.Server, l net.Listener) {
			defer wg.Done()
			slog.Info("Starting server", "address", l.Addr())
			if err := s.Serve(l); !errors.Is(err, http.ErrServerClosed) {
				slog.Error("Error", "address", l.Addr(), "err", err)
				cancel()
			}
		}(servers[i], listeners[i])

		// Wait for context to cancel and stop server.
		wg.Add(1)
		go func(s *http.Server, l net.Listener) {
			defer wg.Done()
			var err error
			//nolint:gosimple // https://github.com/dominikh/go-tools/issues/503
			for {
				select {
				case <-ctx.Done():
					slog.Info("Stopping server", "address", l.Addr())
					// In production do it with timeout.
					err = s.Shutdown(context.Background())
					if err != nil && !errors.Is(err, http.ErrServerClosed) {
						slog.Error("Failed to shutdown server",
							"err", err, "address", l.Addr())
					}
					return
				}
			}
		}(servers[i], listeners[i])
	}

	// Wait for all servers to exit.
	wg.Wait()
	slog.Info("Server(s) stopped")
}
