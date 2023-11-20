// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build !darwin || ios

package launchd_test

import (
	"errors"
	"syscall"
	"testing"

	"github.com/tprasadtp/go-launchd"
)

func TestListeners(t *testing.T) {
	t.Run("TCPListeners", func(t *testing.T) {
		tcpListeners, err := launchd.Listeners("b39422da-351b-50ad-a7cc-9dea5ae436ea")
		if len(tcpListeners) != 0 {
			t.Errorf("expected no listeners on non-darwin platform")
		}

		if !errors.Is(err, syscall.ENOSYS) {
			t.Errorf("expected error=%s, got=%s", syscall.ENOSYS, err)
		}
	})

	t.Run("UDPListeners", func(t *testing.T) {
		udpListeners, err := launchd.PacketListeners("b39422da-351b-50ad-a7cc-9dea5ae436ea")
		if len(udpListeners) != 0 {
			t.Errorf("expected no listeners on non-darwin platform")
		}

		if !errors.Is(err, syscall.ENOSYS) {
			t.Errorf("expected error=%s, got=%s", syscall.ENOSYS, err)
		}
	})
}
