// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build !darwin || ios

package launchd

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

// Os specific implementation of [Files].
func files(_ string) ([]*os.File, error) {
	return nil, fmt.Errorf("launchd: only supported on macOS: %w", syscall.ENOTSUP)
}

// Os specific implementation of [Listeners].
func listeners(_ string) ([]net.Listener, error) {
	return nil, fmt.Errorf("launchd: only supported on macOS: %w", syscall.ENOTSUP)
}

// Os specific implementation of [PacketListeners].
func packetListeners(_ string) ([]net.PacketConn, error) {
	return nil, fmt.Errorf("launchd: only supported on macOS: %w", syscall.ENOTSUP)
}
