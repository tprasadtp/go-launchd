// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package launchd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
)

// Files returns slice of [*os.File] backed by file descriptors for given socket.
//
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if socket is not found.
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - [syscall.ENOSYS] is returned on non macOS platforms (including iOS).
//
// This must be called exactly once for given socket name. Subsequent calls
// with the same socket name will return [syscall.EALREADY].
func Files(name string) ([]*os.File, error) {
	return files(name)
}

// Listeners returns slice of [net.Listener] for specified TCP/stream socket.
//
// In case of error building listeners, an appropriate error is returned,
// along with a partial list of listeners. It is the responsibility of the caller
// to close the returned non nil listeners whenever required.
//
// Closing returned listeners does not close underlying file descriptor
// and closing files does not affect the listeners.
//
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if socket is not found.
//   - [syscall.ESOCKTNOSUPPORT] is returned if socket is of incorrect type.
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - [syscall.ENOSYS] is returned on non macOS platforms (including iOS).
//
// This must be called exactly once for a given socket name. Subsequent calls
// with the same socket name will return [syscall.EALREADY].
func Listeners(name string) ([]net.Listener, error) {
	files, err := Files(name)
	if err != nil {
		return nil, err
	}

	// ESOCKTNOSUPPORT (94) Socket type not supported
	// EOPNOTSUPP (95)    Operation not supported
	// EPFNOSUPPORT (96)    Protocol family not supported
	listeners := make([]net.Listener, 0, len(files))
	for _, file := range files {
		l, el := net.FileListener(file)
		if el != nil {
			err = errors.Join(err, el)
		} else {
			listeners = append(listeners, l)
		}
	}

	if err != nil {
		return slices.Clip(listeners), fmt.Errorf("launchd: %w", err)
	}
	return slices.Clip(listeners), nil
}

// PacketListeners returns slice of [net.PacketConn] for specified UDP/datagram socket.
//
// In case of error building [net.PacketConn], an appropriate error is returned,
// along with a partial list of [net.PacketConn]. It is the responsibility of the
// caller to close the returned non-nil listeners whenever required.
//
// Closing returned listeners does not close underlying file descriptor
// and closing files does not affect the listeners.
//
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if socket is not found.
//   - [syscall.ESOCKTNOSUPPORT] is returned if socket is of incorrect type.
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - [syscall.ENOSYS] is returned on non macOS platforms (including iOS).
//
// This must be called exactly once for a given socket name. Subsequent calls
// with the same socket name will return [syscall.EALREADY].
func PacketListeners(name string) ([]net.PacketConn, error) {
	return packetListeners(name)
}

// Deprecated: Use [Listeners].
func TCPListeners(name string) ([]net.Listener, error) {
	return Listeners(name)
}

// Deprecated: Use [PacketListeners].
func UDPListeners(name string) ([]net.PacketConn, error) {
	return PacketListeners(name)
}
