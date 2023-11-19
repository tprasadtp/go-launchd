// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build cgo

package launchd_test

import (
	"testing"

	"github.com/tprasadtp/go-launchd/internal/testcgo"
)

func TestCgoTests(t *testing.T) {
	t.Run("0", func(t *testing.T) {
		if v := testcgo.Abs(0); v != 0 {
			t.Errorf("expected=0, got=%d", v)
		}
	})
	t.Run("positive", func(t *testing.T) {
		if v := testcgo.Abs(99); v != 99 {
			t.Errorf("expected=99, got=%d", v)
		}
	})

	t.Run("negative", func(t *testing.T) {
		if v := testcgo.Abs(-99); v != 99 {
			t.Errorf("expected=99, got=%d", v)
		}
	})
}
