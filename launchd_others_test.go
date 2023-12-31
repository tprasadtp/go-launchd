// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build !darwin || ios

package launchd_test

import (
	"testing"

	"github.com/tprasadtp/go-launchd"
)

func TestIsManagedByLaunchd(t *testing.T) {
	v, err := launchd.IsManagedByLaunchd()
	if v {
		t.Errorf("expected true, got false")
	}

	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
}
