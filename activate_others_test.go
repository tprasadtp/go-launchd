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

func TestFiles(t *testing.T) {
	files, err := launchd.Files("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(files) != 0 {
		t.Errorf("expected no files on non-darwin platform")
	}

	if !errors.Is(err, syscall.ENOTSUP) {
		t.Errorf("expected error=%s, got=%s", syscall.ENOTSUP, err)
	}
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("expected error=%s, got=%s", errors.ErrUnsupported, err)
	}
}

func TestListeners(t *testing.T) {
	listeners, err := launchd.Listeners("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(listeners) != 0 {
		t.Errorf("expected no listeners on non-darwin platform")
	}

	if !errors.Is(err, syscall.ENOTSUP) {
		t.Errorf("expected error=%s, got=%s", syscall.ENOTSUP, err)
	}
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("expected error=%s, got=%s", errors.ErrUnsupported, err)
	}
}

func TestPacketListeners(t *testing.T) {
	listeners, err := launchd.PacketListeners("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(listeners) != 0 {
		t.Errorf("expected no listeners on non-darwin platform")
	}

	if !errors.Is(err, syscall.ENOTSUP) {
		t.Errorf("expected error=%s, got=%s", syscall.ENOTSUP, err)
	}
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("expected error=%s, got=%s", errors.ErrUnsupported, err)
	}
}

func TestTCPListeners(t *testing.T) {
	listeners, err := launchd.TCPListeners("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(listeners) != 0 {
		t.Errorf("expected no listeners on non-darwin platform")
	}

	if !errors.Is(err, syscall.ENOTSUP) {
		t.Errorf("expected error=%s, got=%s", syscall.ENOTSUP, err)
	}
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("expected error=%s, got=%s", errors.ErrUnsupported, err)
	}
}

func TestUDPListeners(t *testing.T) {
	listeners, err := launchd.UDPListeners("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(listeners) != 0 {
		t.Errorf("expected no listeners on non-darwin platform")
	}

	if !errors.Is(err, syscall.ENOTSUP) {
		t.Errorf("expected error=%s, got=%s", syscall.ENOTSUP, err)
	}
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("expected error=%s, got=%s", errors.ErrUnsupported, err)
	}
}
