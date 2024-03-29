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
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"text/template"
	"time"

	"github.com/tprasadtp/go-launchd"
)

type TestEvent struct {
	Name    string `json:"name"`
	Success bool   `json:"success,omitempty"`
	Message string `json:"message,omitempty"`
}

type TemplateData struct {
	BundleID                 string
	GoTestServerAddr         string
	GoTestBinary             string
	GoTestName               string
	GoCoverDir               string
	StdoutFile               string
	StderrFile               string
	TCP                      string
	TCPInvalidType           string
	TCPMultiple              string
	TCPDualStackSingleSocket string
	UDP                      string
	UDPInvalidType           string
	UDPMultiple              string
	UDPDualStackSingleSocket string
	UnixStreamSocket         string
	UnixDatagramSocket       string
}

//go:embed internal/testdata/launchd.plist
var plistTemplate string

var (
	goCoverDirCache  string
	testCoverDirOnce sync.Once
)

// TestingCoverDir coverage data directory. Returns empty if coverage is not
// enabled or if test.gocoverdir flag or GOCOVERDIR env variable is not specified.
// because tests can enable this globally, it is always resolved to absolute path.
//
// This uses Undocumented/Unexported test flag: -test.gocoverdir.
// https://github.com/golang/go/issues/51430#issuecomment-1344711300
func TestingCoverDir(t *testing.T) string {
	testCoverDirOnce.Do(func() {
		// The return value will be empty if test coverage is not enabled.
		if testing.CoverMode() == "" {
			return
		}

		var goCoverDir string
		gocoverdirFlag := flag.Lookup("test.gocoverdir")
		if goCoverDir == "" && gocoverdirFlag != nil {
			goCoverDir = gocoverdirFlag.Value.String()
		}

		goCoverDirEnv := strings.TrimSpace(os.Getenv("GOCOVERDIR"))
		if goCoverDir == "" && goCoverDirEnv != "" {
			goCoverDir = goCoverDirEnv
		}

		// Return empty string
		if goCoverDir != "" {
			goCoverDirCache = goCoverDir
		}
	})

	if goCoverDirCache == "" {
		return ""
	}

	// Get absolute path for GoCoverDir.
	goCoverDirAbs, err := filepath.Abs(goCoverDirCache)
	if err != nil {
		t.Fatalf("Failed to get absolute path of test.gocoverdir(%s):%s",
			goCoverDirCache, err)
	}
	return goCoverDirAbs
}

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

// NewOutputLogger Creates [OutputLogger].
func NewOutputLogger(t *testing.T, prefix string) *OutputLogger {
	v := &OutputLogger{
		t:      t,
		prefix: prefix,
		buf:    make([]byte, 0, 1024),
	}
	if v.prefix == "" {
		v.prefix = "output"
	}
	return v
}

// OutputLogger writes to t.Log when new lines are found.
type OutputLogger struct {
	t      *testing.T
	buf    []byte
	prefix string
}

func (l *OutputLogger) LogOutput(b []byte) {
	if len(b) == 0 {
		return
	}
	l.t.Helper()
	l.buf = append(l.buf, b...)
	var n int
	for {
		n = bytes.IndexByte(l.buf, '\n')
		if n < 0 {
			break
		}

		l.t.Logf("(%s) %s", l.prefix, l.buf[:n])
		if n+1 > len(l.buf) {
			l.buf = l.buf[0:]
		} else {
			l.buf = l.buf[n+1:]
		}
	}
}

func (l *OutputLogger) Logf(format string, args ...any) {
	l.t.Helper()
	l.t.Logf(format, args...)
}

func (l *OutputLogger) Errorf(format string, args ...any) {
	l.t.Helper()
	l.t.Errorf(format, args...)
}

func (l *OutputLogger) Write(b []byte) (int, error) {
	l.LogOutput(b)
	return len(b), nil
}

// Push events to test server.
func NotifyTestServer(t *testing.T, event TestEvent) {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Errorf("%s", err)
	}

	request, err := http.NewRequestWithContext(
		context.Background(),
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

// Start a simple http server binding to socket and test if it is reachable.
func StreamSocketServerPing(t *testing.T, listener net.Listener, unix bool) {
	t.Helper()
	t.Logf("Listener: %s", listener.Addr())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Logf("StreamSocketServer, method=%s, url=%s, host=%s", r.Method, r.URL, r.Host)
		if r.Method == http.MethodDelete {
			cancel()
		}
	})

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Second * 30,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.Logf("Starting server on launchd socket: %s", listener.Addr())
		if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to listen on %s: %s", listener.Addr(), err)
			NotifyTestServer(t, TestEvent{
				Name:    t.Name(),
				Success: false,
				Message: fmt.Sprintf("Failed to listen on %s: %s", listener.Addr(), err),
			})
			cancel()
		}
	}()

	// Wait for context to be cancelled and shut down the server.
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		//nolint:gosimple // https://github.com/dominikh/go-tools/issues/503
		for {
			select {
			case <-ctx.Done():
				t.Logf("Stopping socket server: %s", listener.Addr())
				err = server.Shutdown(context.Background())
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					t.Errorf("Failed to stop socket server: %s", listener.Addr())
				}
				return
			}
		}
	}()

	var url string
	if unix {
		url = "http://unix"
	} else {
		url = fmt.Sprintf("http://%s", listener.Addr())
	}

	// Try to send HTTP request to socket server.
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		url,
		nil)
	if err != nil {
		NotifyTestServer(t, TestEvent{
			Name:    t.Name(),
			Message: fmt.Sprintf("Failed to build HTTP request: %s", err),
		})
		t.Errorf("Failed to build HTTP request: %s", err)
		cancel()
		wg.Wait()
		return
	}
	client := &http.Client{}
	if unix {
		t.Logf("Using UNIX socket: %s", listener.Addr())
		dialer := &net.Dialer{}
		client.Transport = &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return dialer.DialContext(ctx, "unix", listener.Addr().String())
			},
		}
	} else {
		t.Logf("Using TCP socket: %s", listener.Addr())
	}
	response, err := client.Do(request)
	if err != nil {
		NotifyTestServer(t, TestEvent{
			Name:    t.Name(),
			Message: fmt.Sprintf("Failed to do HTTP request: %s", err),
		})
		t.Errorf("Failed to do HTTP request: %s", err)
		return
	}
	if response != nil {
		if response.Body != nil {
			defer response.Body.Close()
		}
	}

	if response.StatusCode == http.StatusOK {
		NotifyTestServer(t, TestEvent{
			Name:    t.Name(),
			Success: true,
		})
	} else {
		NotifyTestServer(t, TestEvent{
			Name:    t.Name(),
			Message: fmt.Sprintf("Failed to do HTTP request: %s", response.Status),
		})
		t.Errorf("Failed to do HTTP request: %s", response.Status)
	}

	t.Logf("Waiting for socket server to stop...")
	wg.Wait()
}

// DeferCloseListeners.
func DeferCloseListeners(t *testing.T, listeners []net.Listener) {
	t.Helper()
	if len(listeners) > 0 {
		t.Cleanup(func() {
			for _, item := range listeners {
				t.Logf("Closing listener(stream): %s", item.Addr())
				item.Close()
			}
		})
	}
}

// DeferClosePacketListeners.
func DeferClosePacketListeners(t *testing.T, listeners []net.PacketConn) {
	t.Helper()
	if len(listeners) > 0 {
		t.Cleanup(func() {
			for _, item := range listeners {
				t.Logf("Closing listener(datagram): %s", item.LocalAddr())
				item.Close()
			}
		})
	}
}

// TestRemote runs tests and pushes the results to GO_TEST_SERVER_ADDR.
func TestRemote(t *testing.T) {
	if _, ok := os.LookupEnv("GO_TEST_SERVER_ADDR"); !ok {
		t.SkipNow()
	}

	t.Logf("Getwd:%s", func() string {
		v, err := os.Getwd()
		if err != nil {
			return err.Error()
		}
		return v
	}())

	t.Logf("GOCOVERDIR=%s", TestingCoverDir(t))
	t.Logf("Args=%s", os.Args)

	t.Run("Listeners", func(t *testing.T) {
		t.Run("NoSuchSocket", func(t *testing.T) {
			_, err := launchd.Listeners("z")
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
			l, err := launchd.Listeners("tcp")
			DeferCloseListeners(t, l)
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
				t.Run("StreamSocketServerPing", func(t *testing.T) {
					StreamSocketServerPing(t, l[0], false)
				})
			}
		})

		t.Run("ActivateMultipleTimesMustError", func(t *testing.T) {
			_, err := launchd.Listeners("tcp")
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
			l, err := launchd.Listeners("tcp-multiple")
			DeferCloseListeners(t, l)
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
				for i, item := range l {
					t.Run(fmt.Sprintf("StreamSocketServerPing-%d", i+1),
						func(t *testing.T) {
							StreamSocketServerPing(t, item, false)
						})
				}
			}
		})

		t.Run("TCPDualStack-SingleSocket", func(t *testing.T) {
			l, err := launchd.Listeners("tcp-dualstack-single-socket")
			DeferCloseListeners(t, l)
			if err != nil || len(l) != 1 {
				if err != nil {
					event := TestEvent{
						Name:    t.Name() + "ErrorCheck",
						Success: false,
						Message: fmt.Sprintf("expected no error, got=%s", err),
					}
					NotifyTestServer(t, event)
					t.Errorf("expected=nil, got=%s", err)
				}
				if len(l) != 1 {
					event := TestEvent{
						Name:    t.Name(),
						Success: false,
						Message: fmt.Sprintf("expected listeners=1, got=%d", len(l)),
					}
					t.Errorf("expected listeners=1, got=%d", len(l))
					NotifyTestServer(t, event)
				}
			} else {
				t.Run("StreamSocketServerPing", func(t *testing.T) {
					StreamSocketServerPing(t, l[0], false)
				})
			}
		})
		t.Run("TCPInvalidType", func(t *testing.T) {
			l, err := launchd.Listeners("tcp-invalid-type")
			DeferCloseListeners(t, l)
			if !errors.Is(err, syscall.ESOCKTNOSUPPORT) {
				event := TestEvent{
					Name:    t.Name(),
					Success: false,
					Message: fmt.Sprintf("expected error=%s, got=%s", syscall.ESOCKTNOSUPPORT, err),
				}
				NotifyTestServer(t, event)
				t.Errorf("expected error=%s, got=%s", syscall.ESOCKTNOSUPPORT, err)
			} else {
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})

		t.Run("UnixSocket", func(t *testing.T) {
			l, err := launchd.Listeners("unix-stream")
			DeferCloseListeners(t, l)
			if err != nil || len(l) != 1 {
				if err != nil {
					event := TestEvent{
						Name:    t.Name() + "ErrorCheck",
						Success: false,
						Message: fmt.Sprintf("expected no error, got=%s", err),
					}
					NotifyTestServer(t, event)
					t.Errorf("expected=nil, got=%s", err)
				}
				if len(l) != 1 {
					event := TestEvent{
						Name:    t.Name(),
						Success: false,
						Message: fmt.Sprintf("expected listeners=1, got=%d", len(l)),
					}
					t.Errorf("expected listeners=1, got=%d", len(l))
					NotifyTestServer(t, event)
				}
			} else {
				t.Run("StreamSocketServerPing", func(t *testing.T) {
					StreamSocketServerPing(t, l[0], true)
				})
			}
		})
	})

	t.Run("PacketListeners", func(t *testing.T) {
		t.Run("NoSuchSocket", func(t *testing.T) {
			_, err := launchd.PacketListeners("z")
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
			l, err := launchd.PacketListeners("udp")
			DeferClosePacketListeners(t, l)
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
			_, err := launchd.PacketListeners("tcp")
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
			l, err := launchd.PacketListeners("udp-multiple")
			DeferClosePacketListeners(t, l)
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
		t.Run("UDPDualStack-SingleSocket", func(t *testing.T) {
			l, err := launchd.PacketListeners("udp-dualstack-single-socket")
			DeferClosePacketListeners(t, l)
			if err != nil || len(l) != 1 {
				if err != nil {
					event := TestEvent{
						Name:    t.Name() + "ErrorCheck",
						Success: false,
						Message: fmt.Sprintf("expected no error, got=%s", err),
					}
					NotifyTestServer(t, event)
					t.Errorf("expected=nil, got=%s", err)
				}
				if len(l) != 1 {
					event := TestEvent{
						Name:    t.Name(),
						Success: false,
						Message: fmt.Sprintf("expected listeners=1, got=%d", len(l)),
					}
					t.Errorf("expected listeners=1, got=%d", len(l))
					NotifyTestServer(t, event)
				}
			} else {
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})

		t.Run("UDPInvalidType", func(t *testing.T) {
			l, err := launchd.PacketListeners("udp-invalid-type")
			DeferClosePacketListeners(t, l)
			if !errors.Is(err, syscall.ESOCKTNOSUPPORT) {
				event := TestEvent{
					Name:    t.Name(),
					Success: false,
					Message: fmt.Sprintf("expected error=%s, got=%s", syscall.ESOCKTNOSUPPORT, err),
				}
				NotifyTestServer(t, event)
				t.Errorf("expected error=%s, got=%s", syscall.ESOCKTNOSUPPORT, err)
			} else {
				event := TestEvent{Name: t.Name(), Success: true}
				NotifyTestServer(t, event)
			}
		})

		t.Run("UnixDatagramSocket", func(t *testing.T) {
			l, err := launchd.PacketListeners("unix-datagram")
			DeferClosePacketListeners(t, l)
			if err != nil || len(l) != 1 {
				if err != nil {
					event := TestEvent{
						Name:    t.Name() + "ErrorCheck",
						Success: false,
						Message: fmt.Sprintf("expected no error, got=%s", err),
					}
					NotifyTestServer(t, event)
					t.Errorf("expected=nil, got=%s", err)
				}
				if len(l) != 1 {
					event := TestEvent{
						Name:    t.Name(),
						Success: false,
						Message: fmt.Sprintf("expected listeners=1, got=%d", len(l)),
					}
					t.Errorf("expected listeners=1, got=%d", len(l))
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

func TestLaunchd(t *testing.T) {
	counter := struct {
		ok       atomic.Uint64
		err      atomic.Uint64
		showLogs atomic.Bool
	}{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)

	t.Logf("Getwd:%s", func() string {
		v, err := os.Getwd()
		if err != nil {
			return err.Error()
		}
		return v
	}())

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
	//
	bundle := fmt.Sprintf("test.go-svc.%s", hex.EncodeToString(rb))

	plistFileName := filepath.Join(agentsDir, fmt.Sprintf("%s.plist", bundle))
	data := TemplateData{
		BundleID:                 bundle,
		GoTestServerAddr:         server.URL,
		GoTestBinary:             os.Args[0],
		GoTestName:               "^(TestRemote|TestTrampoline)",
		GoCoverDir:               TestingCoverDir(t),
		StdoutFile:               stdout,
		StderrFile:               stderr,
		TCP:                      strconv.Itoa(GetFreePort(t)),
		TCPInvalidType:           strconv.Itoa(GetFreePort(t)),
		UDP:                      strconv.Itoa(GetFreePort(t)),
		UDPInvalidType:           strconv.Itoa(GetFreePort(t)),
		TCPMultiple:              strconv.Itoa(GetFreePort(t)),
		UDPMultiple:              strconv.Itoa(GetFreePort(t)),
		TCPDualStackSingleSocket: strconv.Itoa(GetFreePort(t)),
		UDPDualStackSingleSocket: strconv.Itoa(GetFreePort(t)),
		UnixStreamSocket:         filepath.Join(dir, "unix-stream.socket"),
		UnixDatagramSocket:       filepath.Join(dir, "unix-datagram.socket"),
	}

	t.Logf("GoCoverDir=%s", data.GoCoverDir)
	t.Logf("Ports: TCP=%s, TCPDualStack=%s, TCPDualStackSingleSocket=%s TCPInvalidType=%s",
		data.TCP, data.TCPMultiple, data.TCPDualStackSingleSocket, data.TCPInvalidType)
	t.Logf("Ports: UDP=%s, UDPDualStack=%s, UDPDualStackSingleSocket=%s, UDPInvalidType=%s",
		data.UDP, data.UDPMultiple, data.UDPDualStackSingleSocket, data.UDPInvalidType)
	t.Logf("UnixStreamSocket=%s", data.UnixStreamSocket)
	t.Logf("UnixDatagramSocket=%s", data.UnixDatagramSocket)

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
		t.Errorf("Test timed out while waiting for remote (remote panic?)")
	}

	t.Logf("errors=%d, ok=%d, logs=%t", counter.err.Load(), counter.ok.Load(), counter.showLogs.Load())

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
	t.Logf("Output from launch unit: %s", bundle)

	t.Logf("Reading stdout from %s", stdout)
	stdoutBuf, err := os.ReadFile(stdout)
	if err != nil {
		t.Errorf("Failed to read output from stdout: %s", err)
	}
	NewOutputLogger(t, "Remote Stdout").LogOutput(stdoutBuf)

	t.Logf("Reading stderr from %s", stderr)
	stderrBuf, err := os.ReadFile(stderr)
	if err != nil {
		t.Errorf("Failed to read output from stderr: %s", err)
	}
	NewOutputLogger(t, "Remote Stderr").LogOutput(stderrBuf)
}

func TestListeners_NotManagedByLaunchd(t *testing.T) {
	rv, err := launchd.Listeners("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(rv) != 0 {
		t.Errorf("expected no listeners when process is not manged by launchd")
	}
	if !errors.Is(err, syscall.ESRCH) {
		t.Errorf("expected error=%s, got=%s", syscall.Errno(3), err)
	}
}

func TestPacketListeners_NotManagedByLaunchd(t *testing.T) {
	rv, err := launchd.PacketListeners("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(rv) != 0 {
		t.Errorf("expected no listeners when process is not manged by launchd")
	}
	if !errors.Is(err, syscall.ESRCH) {
		t.Errorf("expected error=%s, got=%s", syscall.Errno(3), err)
	}
}
