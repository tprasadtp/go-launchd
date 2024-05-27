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

type testEvent struct {
	Name    string `json:"name"`
	Success bool   `json:"success,omitempty"`
	Message string `json:"message,omitempty"`
}

type templateData struct {
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

//nolint:gochecknoglobals
var (
	goCoverDirCache  string
	testCoverDirOnce sync.Once
)

// coverageDir coverage data directory. Returns empty if coverage is not
// enabled or if test.gocoverdir flag or GOCOVERDIR env variable is not specified.
// because tests can enable this globally, it is always resolved to absolute path.
//
// This uses unexported test flag: -test.gocoverdir.
// https://github.com/golang/go/issues/51430#issuecomment-1344711300
func coverageDir(tb testing.TB) string {
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
		tb.Fatalf("Failed to get absolute path of test.gocoverdir(%s):%s",
			goCoverDirCache, err)
	}
	return goCoverDirAbs
}

// getFreePort asks the kernel for a free open port that is ready to use.
func getFreePort(t *testing.T) int {
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

var _ io.Writer = (*writer)(nil)

// NewTestingWriter returns an [io.Writer] which writes to [testing.TB.Log],
// Optionally with a prefix. Only handles unix new lines.
func NewTestingWriter(tb testing.TB, prefix string) io.Writer {
	return &writer{
		tb:     tb,
		prefix: prefix,
		buf:    make([]byte, 0, 1024),
	}
}

// Writes to t.Log when new lines are found.
type writer struct {
	prefix string
	tb     testing.TB
	buf    []byte
}

func (l *writer) Write(b []byte) (int, error) {
	l.buf = append(l.buf, b...)
	var n int
	for {
		n = bytes.IndexByte(l.buf, '\n')
		if n < 0 {
			break
		}

		if l.prefix != "" {
			l.tb.Logf("(%s) %s", l.prefix, l.buf[:n])
		} else {
			l.tb.Log(string(l.buf[:n]))
		}

		if n+1 > len(l.buf) {
			l.buf = l.buf[0:]
		} else {
			l.buf = l.buf[n+1:]
		}
	}
	return len(b), nil
}

// Push events to test server.
func notifyTestServer(t *testing.T, ok bool, msg string) {
	t.Helper()
	event := testEvent{
		Name:    t.Name(),
		Success: ok,
		Message: msg,
	}
	body, err := json.Marshal(event)
	if err != nil {
		t.Errorf("%s", err)
		return
	}

	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		os.Getenv("GO_TEST_SERVER_ADDR"),
		bytes.NewReader(body))
	if err != nil {
		t.Errorf("%s", err)
		return
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := client.Do(request)
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	defer resp.Body.Close()
}

// Start a simple http server binding to socket and test if it is reachable.
func streamServerPing(t *testing.T, listener net.Listener) {
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

	var w sync.WaitGroup
	w.Add(1)
	go func() {
		defer w.Done()
		t.Logf("Starting server on launchd socket: %s", listener.Addr())
		if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			msg := fmt.Sprintf("Failed to listen on %s: %s", listener.Addr(), err)
			t.Error(msg)
			notifyTestServer(t, false, msg)
			cancel()
		}
	}()

	// Wait for context to be cancelled and shut down the server.
	w.Add(1)
	go func() {
		defer w.Done()
		var err error
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
	_, isUnixListener := listener.(*net.UnixListener)

	if isUnixListener {
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
		msg := fmt.Sprintf("Failed to build HTTP request: %s", err)
		notifyTestServer(t, false, msg)
		t.Error(msg)
		cancel()
		w.Wait()
		return
	}
	client := &http.Client{}
	if isUnixListener {
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
		msg := fmt.Sprintf("Failed to do HTTP request: %s", err)
		t.Errorf(msg)
		notifyTestServer(t, false, msg)
		return
	}
	if response != nil {
		if response.Body != nil {
			defer response.Body.Close()
		}
	}

	if response.StatusCode == http.StatusOK {
		notifyTestServer(t, true, "")
	} else {
		msg := fmt.Sprintf("Failed to do HTTP request: %s", response.Status)
		t.Error(msg)
		notifyTestServer(t, true, "")
	}

	t.Logf("Waiting for socket server to stop...")
	w.Wait()
}

// cleanupNetListeners.
func cleanupNetListeners(t *testing.T, listeners []net.Listener) {
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

// cleanupPacketListeners.
func cleanupPacketListeners(t *testing.T, listeners []net.PacketConn) {
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
	if v, ok := os.LookupEnv("GO_TEST_SERVER_ADDR"); !ok {
		t.SkipNow()
	} else {
		t.Logf("GO_TEST_SERVER_ADDR=%s", v)
	}

	t.Logf("GOCOVERDIR=%s", coverageDir(t))
	t.Logf("Args=%s", os.Args)

	tt := []struct {
		name   string
		socket string
		errs   []error
		count  int
		dgram  bool
	}{
		{
			name:   "TCP-NoSuchSocket",
			socket: "5bf300ce-6993-4fd5-bfa9-bc1c9e49f996",
			errs:   []error{syscall.ENOENT, syscall.ESRCH},
		},
		{
			name:   "TCP-SingleSocket",
			socket: "tcp",
			count:  1,
		},
		{
			name:   "TCP-ActivateMultipleTimesMustError",
			socket: "tcp",
			errs:   []error{syscall.EALREADY},
		},
		{
			name:   "TCP-MultipleSockets",
			socket: "tcp-multiple",
			count:  2, // one for ipv6 and ipv4
		},
		{
			name:   "TCP-DualStack-SingleSocket",
			socket: "tcp-dualstack-single-socket",
			count:  1,
		},
		{
			name:   "TCP-InvalidType",
			socket: "tcp-invalid-type",
			errs:   []error{syscall.ESOCKTNOSUPPORT},
		},
		{
			name:   "UnixStreamSocket",
			socket: "unix-stream",
			count:  1,
		},
		// UDP/Stream sockets.
		{
			name:   "UDP-NoSuchSocket",
			socket: "9f712891-ca0b-4de7-8750-645c74008ecd",
			errs:   []error{syscall.ENOENT, syscall.ESRCH},
			dgram:  true,
		},
		{
			name:   "UDP-SingleSocket",
			socket: "udp",
			count:  1,
			dgram:  true,
		},
		{
			name:   "UDP-ActivateMultipleTimesMustError",
			socket: "udp",
			errs:   []error{syscall.EALREADY},
			dgram:  true,
		},
		{
			name:   "UDP-MultipleSockets",
			socket: "udp-multiple",
			count:  2, // one for ipv6 and ipv4
			dgram:  true,
		},
		{
			name:   "UDP-DualStack-SingleSocket",
			socket: "udp-dualstack-single-socket",
			count:  1,
			dgram:  true,
		},
		{
			name:   "UDP-InvalidType",
			socket: "udp-invalid-type",
			errs:   []error{syscall.ESOCKTNOSUPPORT},
			dgram:  true,
		},
		{
			name:   "UnixDatagramSocket",
			socket: "unix-datagram",
			count:  1,
			dgram:  true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var listeners []net.Listener
			var packetListeners []net.PacketConn
			var listenerCount int
			var err error

			if tc.dgram {
				packetListeners, err = launchd.PacketListeners(tc.socket)
				listenerCount = len(packetListeners)
				cleanupPacketListeners(t, packetListeners)
			} else {
				listeners, err = launchd.Listeners(tc.socket)
				listenerCount = len(listeners)
				cleanupNetListeners(t, listeners)
			}

			// Check if error is one of specified or nil.
			t.Run("CheckError", func(t *testing.T) {
				if len(tc.errs) > 0 {
					ok := false
					for i := range tc.errs {
						if errors.Is(err, tc.errs[i]) {
							ok = true
							break
						}
					}
					if !ok {
						msg := fmt.Sprintf("expected error(%v), but got=%s", tc.errs, err)
						t.Error(msg)
						notifyTestServer(t, false, msg)
					} else {
						notifyTestServer(t, true, "")
					}
				} else {
					if err != nil {
						msg := fmt.Sprintf("expected no error, but got=%s", err)
						t.Error(msg)
						notifyTestServer(t, false, msg)
					} else {
						notifyTestServer(t, true, "")
					}
				}
			})

			// Check listener count.
			t.Run("ListenerCount", func(t *testing.T) {
				if listenerCount != tc.count {
					msg := fmt.Sprintf("expected listeners=%d, but got=%d", tc.count, listenerCount)
					t.Error(msg)
					notifyTestServer(t, false, msg)
				} else {
					notifyTestServer(t, true, "")
				}
			})

			// Ensure listening on the steam socket works.
			if len(listeners) > 0 {
				for i := range listeners {
					t.Run(fmt.Sprintf("ServerPing-%d", i+1), func(t *testing.T) {
						streamServerPing(t, listeners[i])
					})
				}
			}
		})
	}

	// notify test server.
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
		ok  atomic.Uint64
		err atomic.Uint64
	}{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)

	// Handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			b, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				t.Errorf("Error reading request: %s", err)
				return
			}
			var event testEvent
			err = json.Unmarshal(b, &event)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
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
		err = os.MkdirAll(agentsDir, 0o700)
		if err != nil {
			t.Fatalf("Failed to create dir: %s", err)
		}
	}

	// Generate random prefix for test.
	rb := make([]byte, 9)
	_, err = rand.Read(rb)
	if err != nil {
		t.Fatalf("Failed to generate random bundle suffix")
	}

	coverageDir := coverageDir(t)
	if coverageDir == "" {
		coverageDir = t.TempDir()
	}

	// Render template.
	bundle := fmt.Sprintf("test.go-svc.%s", hex.EncodeToString(rb))
	plistFileName := filepath.Join(agentsDir, fmt.Sprintf("%s.plist", bundle))
	data := templateData{
		BundleID:                 bundle,
		GoTestServerAddr:         server.URL,
		GoTestBinary:             os.Args[0],
		GoTestName:               "^TestRemote",
		GoCoverDir:               coverageDir,
		StdoutFile:               stdout,
		StderrFile:               stderr,
		TCP:                      strconv.Itoa(getFreePort(t)),
		TCPInvalidType:           strconv.Itoa(getFreePort(t)),
		UDP:                      strconv.Itoa(getFreePort(t)),
		UDPInvalidType:           strconv.Itoa(getFreePort(t)),
		TCPMultiple:              strconv.Itoa(getFreePort(t)),
		UDPMultiple:              strconv.Itoa(getFreePort(t)),
		TCPDualStackSingleSocket: strconv.Itoa(getFreePort(t)),
		UDPDualStackSingleSocket: strconv.Itoa(getFreePort(t)),
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
	if len(output) > 0 {
		t.Logf("launchctl load output: %s", string(output))
	}
	if err != nil {
		t.Fatalf("Failed to load plist: %s", err)
	}
	t.Cleanup(func() {
		t.Logf("Unloading plist file: %s", plistFileName)
		cmd = exec.Command("launchctl", "unload", plistFileName)
		output, err = cmd.CombinedOutput()
		if len(output) > 0 {
			t.Logf("launchctl unload output: %s", string(output))
		}
		if err != nil {
			t.Fatalf("Failed to unload plist: %s", err)
		}
	})

	// Waiting for test binary to POST results
	t.Logf("Waiting for remote tests to publish results...")
	select {
	case <-ctx.Done():
	}

	// Check if test timed out
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Errorf("Test timed out while waiting for remote (remote panic?)")
	}

	t.Logf("Remote test counters errors=%d, ok=%d", counter.err.Load(), counter.ok.Load())

	// Check if Test results.
	switch {
	case counter.err.Load() == 0 && counter.ok.Load() == 0:
		t.Errorf("Remote test did not post its results")
	case counter.err.Load() == 0 && counter.ok.Load() > 1:
		t.Logf("%d Remote tests successful", counter.ok.Load())
	default:
		t.Errorf("%d Remote tests returned errors", counter.err.Load())
	}

	// Check Log output from launchd unit
	t.Logf("Output from launch unit: %s", bundle)

	t.Logf("Reading stdout from %s", stdout)
	stdoutBuf, err := os.ReadFile(stdout)
	if err != nil {
		t.Errorf("Failed to read output from stdout: %s", err)
	}
	remoteOutputWriter := NewTestingWriter(t, "Remote Stdout")
	_, _ = remoteOutputWriter.Write(stdoutBuf)
	_, _ = remoteOutputWriter.Write(nil) // flush any pending buffers.

	t.Logf("Reading stderr from %s", stderr)
	stderrBuf, err := os.ReadFile(stderr)
	if err != nil {
		t.Errorf("Failed to read output from stderr: %s", err)
	}
	remoteErrWriter := NewTestingWriter(t, "Remote Stderr")
	_, _ = remoteErrWriter.Write(stderrBuf)
	_, _ = remoteErrWriter.Write(nil) // flush any pending buffers.
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
