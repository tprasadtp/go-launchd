// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build darwin && !ios

package launchd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type TestEvent struct {
	Name    string `json:"name"`
	Success bool   `json:"success,omitempty"`
	Message string `json:"message,omitempty"`
}

// TestingCoverDir coverage data directory. Returns empty if coverage is not
// enabled or if test.gocoverdir flag or GOCOVERDIR env variable is not specified.
//
// This uses Undocumented/Unexported test flag: -test.gocoverdir.
// https://github.com/golang/go/issues/51430#issuecomment-1344711300
func TestingCoverDir(t *testing.T) string {
	t.Helper()

	// The return value will be empty if test coverage is not enabled.
	if testing.CoverMode() != "" {
		return ""
	}

	var goCoverDir string
	var gocoverdirFlag = flag.Lookup("test.gocoverdir")
	if goCoverDir == "" && gocoverdirFlag != nil {
		goCoverDir = gocoverdirFlag.Value.String()
	}

	var goCoverDirEnv = strings.TrimSpace(os.Getenv("GOCOVERDIR"))
	if goCoverDir == "" && goCoverDirEnv != "" {
		goCoverDir = goCoverDirEnv
	}

	// Return empty string
	if goCoverDir == "" {
		return ""
	}

	// Get absolute path for GoCoverDir.
	// Because launchd unit may run under different working directory.
	goCoverDirAbs, err := filepath.Abs(goCoverDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path of test.gocoverdir(%s):%s",
			goCoverDir, err)
	}
	return goCoverDirAbs
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
