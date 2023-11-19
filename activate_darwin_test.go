// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build darwin && !ios

package launchd_test

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"testing"
	"text/template"
	"time"

	"github.com/tprasadtp/go-launchd"
)

type TestEvent struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type TemplateData struct {
	BundleID         string
	GoTestServerAddr string
	GoTestBinary     string
	GoTestName       string
	StdoutFile       string
	StderrFile       string
	TCP              string
	TCPMultiple      string
	UDP              string
	UDPMultiple      string
}

//go:embed internal/testdata/launchd.plist
var plistTemplate string

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort(t *testing.T) int {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to get free port: %s", err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("failed to get free port: %s", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// Push events to test server.
func NotifyTestServer(t *testing.T, event TestEvent) {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Errorf("%s", err)
	}

	ctx := context.Background()
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		os.Getenv("GO_TEST_SERVER_ADDR"),
		bytes.NewReader(body))
	if err != nil {
		t.Errorf("%s", err)
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := client.Do(request)
	if err != nil {
		t.Errorf("%s", err)
	}
	defer resp.Body.Close()
}

// TestRemote runs tests and pushes the results to GO_TEST_SERVER_ADDR.
func TestRemote(t *testing.T) {
	if _, ok := os.LookupEnv("GO_TEST_SERVER_ADDR"); !ok {
		t.SkipNow()
	}

	t.Run("TCPListenersWithName", func(t *testing.T) {
		t.Run("NoSuchSocket", func(t *testing.T) {
			_, err := launchd.TCPListenersWithName("z")
			// As per docs, it should be ENOENT, but it returns ESRCH.
			if !errors.Is(err, syscall.ENOENT) && !errors.Is(err, syscall.ESRCH) {
				event := TestEvent{
					Name:    t.Name(),
					Success: false,
					Message: fmt.Sprintf("expected=%s, got=%s", syscall.ENOENT, err),
				}
				NotifyTestServer(t, event)
				t.Errorf("expected=%s, got=%s", syscall.ENOENT, err)
			} else {
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})

		t.Run("SingleSocket", func(t *testing.T) {
			l, err := launchd.TCPListenersWithName("tcp")
			if len(l) > 0 {
				t.Cleanup(func() {
					for _, item := range l {
						item.Close()
					}
				})
			}
			if err != nil || len(l) < 1 {
				if err != nil {
					event := TestEvent{
						Name:    t.Name() + "-ErrorCheck",
						Success: false,
						Message: fmt.Sprintf("expected no error, got=%s", err),
					}
					NotifyTestServer(t, event)
					t.Errorf("expected=nil, got=%s", err)
				}
				if len(l) == 0 {
					event := TestEvent{
						Name:    t.Name(),
						Success: false,
						Message: fmt.Sprintf("expected listeners>0, got=%d", len(l)),
					}
					t.Errorf("expected listeners>0, got=%d", len(l))
					NotifyTestServer(t, event)
				}
			} else {
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})

		t.Run("ActivateMultipleTimesMustError", func(t *testing.T) {
			_, err := launchd.TCPListenersWithName("tcp")
			if !errors.Is(err, syscall.EALREADY) {
				event := TestEvent{
					Name:    t.Name(),
					Success: false,
					Message: fmt.Sprintf("expected error=%s, got=%s", syscall.EALREADY, err),
				}
				NotifyTestServer(t, event)
				t.Errorf("expected error=%s, got=%s", syscall.EALREADY, err)
			} else {
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})

		t.Run("TCPMultiple", func(t *testing.T) {
			l, err := launchd.TCPListenersWithName("tcp-multiple")
			if len(l) > 0 {
				t.Cleanup(func() {
					for _, item := range l {
						item.Close()
					}
				})
			}
			if err != nil || len(l) < 2 {
				if err != nil {
					event := TestEvent{
						Name:    t.Name() + "ErrorCheck",
						Success: false,
						Message: fmt.Sprintf("expected no error, got=%s", err),
					}
					NotifyTestServer(t, event)
					t.Errorf("expected=nil, got=%s", err)
				}
				if len(l) < 2 {
					event := TestEvent{
						Name:    t.Name(),
						Success: false,
						Message: fmt.Sprintf("expected listeners>1, got=%d", len(l)),
					}
					t.Errorf("expected listeners>1, got=%d", len(l))
					NotifyTestServer(t, event)
				}
			} else {
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})
	})

	t.Run("UDPListenersWithName", func(t *testing.T) {
		t.Run("NoSuchSocket", func(t *testing.T) {
			_, err := launchd.UDPListenersWithName("z")
			// As per docs, it should be ENOENT, but it returns ESRCH.
			if !errors.Is(err, syscall.ENOENT) && !errors.Is(err, syscall.ESRCH) {
				event := TestEvent{
					Name:    t.Name(),
					Success: false,
					Message: fmt.Sprintf("expected=%s, got=%s", syscall.ENOENT, err),
				}
				NotifyTestServer(t, event)
				t.Errorf("expected=%s, got=%s", syscall.ENOENT, err)
			} else {
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})

		t.Run("SingleSocket", func(t *testing.T) {
			l, err := launchd.UDPListenersWithName("udp")
			if len(l) > 0 {
				t.Cleanup(func() {
					for _, item := range l {
						item.Close()
					}
				})
			}
			if err != nil || len(l) < 1 {
				if err != nil {
					event := TestEvent{
						Name:    t.Name() + "-ErrorCheck",
						Success: false,
						Message: fmt.Sprintf("expected no error, got=%s", err),
					}
					NotifyTestServer(t, event)
					t.Errorf("expected=nil, got=%s", err)
				}
				if len(l) == 0 {
					event := TestEvent{
						Name:    t.Name(),
						Success: false,
						Message: fmt.Sprintf("expected listeners>0, got=%d", len(l)),
					}
					t.Errorf("expected listeners>0, got=%d", len(l))
					NotifyTestServer(t, event)
				}
			} else {
				t.Logf("Listener: %+v", l[0])
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})

		t.Run("ActivateMultipleTimesMustError", func(t *testing.T) {
			_, err := launchd.UDPListenersWithName("tcp")
			if !errors.Is(err, syscall.EALREADY) {
				event := TestEvent{
					Name:    t.Name(),
					Success: false,
					Message: fmt.Sprintf("expected error=%s, got=%s", syscall.EALREADY, err),
				}
				NotifyTestServer(t, event)
				t.Errorf("expected error=%s, got=%s", syscall.EALREADY, err)
			} else {
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})

		t.Run("MultipleSockets", func(t *testing.T) {
			l, err := launchd.UDPListenersWithName("udp-multiple")
			if len(l) > 0 {
				t.Cleanup(func() {
					for _, item := range l {
						item.Close()
					}
				})
			}
			if err != nil || len(l) < 2 {
				if err != nil {
					event := TestEvent{
						Name:    t.Name() + "ErrorCheck",
						Success: false,
						Message: fmt.Sprintf("expected no error, got=%s", err),
					}
					NotifyTestServer(t, event)
					t.Errorf("expected=nil, got=%s", err)
				}
				if len(l) < 2 {
					event := TestEvent{
						Name:    t.Name(),
						Success: false,
						Message: fmt.Sprintf("expected listeners>1, got=%d", len(l)),
					}
					t.Errorf("expected listeners>1, got=%d", len(l))
					NotifyTestServer(t, event)
				}
			} else {
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})
	})

	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		os.Getenv("GO_TEST_SERVER_ADDR"),
		nil)
	if err != nil {
		t.Fatalf("%s", err)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer resp.Body.Close()
}

func TestListenersWithName(t *testing.T) {
	counter := struct {
		ok       atomic.Uint64
		err      atomic.Uint64
		showLogs atomic.Bool
	}{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)

	// Handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			b, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				counter.showLogs.Store(true)
				t.Errorf("Error reading request: %s", err)
				return
			}
			var event TestEvent
			err = json.Unmarshal(b, &event)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				counter.showLogs.Store(true)
				t.Errorf("Error unmarshal data: %s", err)
				return
			}

			if event.Success {
				counter.ok.Add(1)
				t.Logf("%s => SUCCESS", event.Name)
			} else {
				counter.err.Add(1)
				t.Logf("%s => ERROR %s", event.Name, event.Message)
			}
		case http.MethodDelete:
			t.Logf("Received all test events")
			cancel()
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			t.Errorf("Unsupported request method: %s", r.Method)
			counter.showLogs.Store(true)
			return
		}
	})
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		t.Logf("Stopping test server %s", server.URL)
		server.Close()
	})
	t.Logf("Test server listening on %s", server.URL)

	// Temporary directory for launchd output files.
	dir := t.TempDir()
	stdout := filepath.Join(dir, "stdout.log")
	stderr := filepath.Join(dir, "stderr.log")

	h, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get UserHomeDir: %s", err)
	}
	agentsDir := filepath.Join(h, "Library", "LaunchAgents")

	// Create launchd directory if not exists.
	if _, err := os.Stat(agentsDir); errors.Is(err, os.ErrNotExist) {
		t.Logf("Creating dir - %s", agentsDir)
		err = os.MkdirAll(agentsDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create dir: %s", err)
		}
	}

	// Generate random prefix for test
	rb := make([]byte, 9)
	_, err = rand.Read(rb)
	if err != nil {
		t.Fatalf("Failed to generate random bundle suffix")
	}

	// Render template
	bundle := fmt.Sprintf("test.go-svc.%s", hex.EncodeToString(rb))
	plistFileName := filepath.Join(agentsDir, fmt.Sprintf("%s.plist", bundle))
	data := TemplateData{
		BundleID:         bundle,
		GoTestServerAddr: server.URL,
		GoTestBinary:     os.Args[0],
		GoTestName:       "^(TestRemote|TestTrampoline)",
		StdoutFile:       stdout,
		StderrFile:       stderr,
		TCP:              strconv.Itoa(GetFreePort(t)),
		UDP:              strconv.Itoa(GetFreePort(t)),
		TCPMultiple:      strconv.Itoa(GetFreePort(t)),
		UDPMultiple:      strconv.Itoa(GetFreePort(t)),
	}

	t.Logf("Ports: TCP=%s, UDP=%s, TCPDualStack=%s, UDPDualStack=%s",
		data.UDP, data.TCP, data.UDPMultiple, data.TCPMultiple)

	t.Logf("Creating plist file: %s", plistFileName)
	plistFile, err := os.OpenFile(plistFileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("failed to create service file: %s", err)
	}
	t.Cleanup(func() {
		t.Logf("Removing plist file: %s", plistFileName)
		err = os.Remove(plistFileName)
		if err != nil {
			t.Errorf("Failed to cleanup plist file %s: %s", plistFileName, err)
		}
	})

	t.Logf("Rendering plist template to: %s", plistFileName)
	tpl, err := template.New("plist.template").Parse(plistTemplate)
	if err != nil {
		t.Fatalf("invalid plist template: %s", err)
	}
	if err := tpl.Execute(plistFile, data); err != nil {
		t.Fatalf("failed to render plist template: %s", err)
	}

	// sync and close plist file.
	err = plistFile.Sync()
	if err != nil {
		t.Fatalf("Failed to sync plist file: %s", err)
	}

	err = plistFile.Close()
	if err != nil {
		t.Fatalf("Failed to close plist file: %s", err)
	}

	// Load Launchd Unit
	t.Logf("Loading plist unit: %s", plistFileName)
	if _, err := exec.LookPath("launchctl"); err != nil {
		t.Fatalf("launchctl binary is not available")
	}
	cmd := exec.CommandContext(ctx, "launchctl", "load", "-w", plistFileName)
	output, err := cmd.CombinedOutput()
	t.Logf("launchctl load output: %s", string(output))
	if err != nil {
		t.Fatalf("Failed to load plist: %s", err)
	}
	t.Cleanup(func() {
		t.Logf("Unloading plist file: %s", plistFileName)
		cmd = exec.Command("launchctl", "unload", plistFileName)
		output, err = cmd.CombinedOutput()
		t.Logf("launchctl unload output: %s", string(output))
		if err != nil {
			t.Fatalf("Failed to unload plist: %s", err)
		}
	})

	// Waiting for test binary to POST results
	t.Logf("Waiting for remote tests to publish results...")
	//nolint:gosimple // ignore
	select {
	case <-ctx.Done():
	}

	// Check if test timed out
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Errorf("Test timed out while waiting for remote to publish results")
	}

	t.Logf("counter.err=%d, ok=%d, logs=%t", counter.err.Load(), counter.ok.Load(), counter.showLogs.Load())

	// Check if Test results.
	switch {
	case counter.err.Load() == 0 && counter.ok.Load() == 0:
		t.Errorf("Remote test did not post its results")
	case counter.err.Load() == 0 && counter.ok.Load() > 1:
		t.Logf("%d Remote tests successful", counter.ok.Load())
	default:
		t.Errorf("%d Remote tests returned an error", counter.err.Load())
	}

	// Check Log output from launchd unit
	buf, _ := os.ReadFile(stdout)
	t.Logf("Remote Stdout:\n%s", string(buf))

	buf, _ = os.ReadFile(stderr)
	t.Logf("Remote Stderr:\n%s", string(buf))
}

func TestTCPListenersWithName_NotManagedByLaunchd(t *testing.T) {
	rv, err := launchd.TCPListenersWithName("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(rv) != 0 {
		t.Errorf("expected no listeners when process is not manged by launchd")
	}
	if !errors.Is(err, syscall.ESRCH) {
		t.Errorf("expected error=%s, got=%s", syscall.Errno(3), err)
	}
}

func TestUDPListenersWithName_NotManagedByLaunchd(t *testing.T) {
	rv, err := launchd.UDPListenersWithName("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(rv) != 0 {
		t.Errorf("expected no listeners when process is not manged by launchd")
	}
	if !errors.Is(err, syscall.ESRCH) {
		t.Errorf("expected error=%s, got=%s", syscall.Errno(3), err)
	}
}
