// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build !darwin || ios

package launchd_test

import (
	"testing"

	"github.com/tprasadtp/go-launchd"
)

func TestListeners(t *testing.T) {
	tcpListeners, err := launchd.TCPListeners("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(tcpListeners) != 0 {
		t.Errorf("expected no listeners on non-darwin platform")
	}

	if err == nil {
		t.Errorf("expected error, got nil")
	}

	udpListeners, err := launchd.UDPListeners("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(udpListeners) != 0 {
		t.Errorf("expected no listeners on non-darwin platform")
	}

	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
