// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build !darwin || ios

package launchd

import (
	"fmt"
	"net"
)

// ListenersWithName is only supported on macOS. On non macOS platforms
// (including) ios, this will always return error.
func ListenersWithName(_ string) ([]net.Listener, error) {
	return nil, fmt.Errorf("launchd: only supported on macOS")
}
