// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build !darwin

package launchd_test

import (
	"testing"

	"github.com/tprasadtp/go-launchd"
)

func TestListenersWithName(t *testing.T) {
	l, err := launchd.ListenersWithName("b39422da-351b-50ad-a7cc-9dea5ae436ea")
	if len(l) != 0 {
		t.Errorf("expected no listeners on non-darwin platform")
	}

	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
