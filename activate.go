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
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if specified socket
//     is not found.
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - [syscall.ENOSYS] is returned on non macOS platforms (including iOS).
func Files(name string) ([]*os.File, error) {
	fdSlice, err := listenerFdsWithName(name)
	if err != nil {
		return nil, err
	}
	files := make([]*os.File, 0, len(fdSlice))
	for _, fd := range fdSlice {
		if fd != 0 {
			files = append(files, os.NewFile(uintptr(fd),
				fmt.Sprintf("launchd-socket://%s", name)))
		}
	}
	return slices.Clip(files), nil
}

// Listeners returns slice of [net.Listener] for specified TCP/stream socket.
//
// In case of error building listeners, an appropriate error is returned,
// along with partial list of listeners. It is responsibility of the caller to
// close the returned returned non nil listeners whenever required.
//
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if specified socket
//     is not found.
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - [syscall.ENOSYS] is returned on non macOS platforms (including iOS).
func Listeners(name string) ([]net.Listener, error) {
	files, err := Files(name)
	if err != nil {
		return nil, err
	}

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
// In case of error building listeners, an appropriate error is returned,
// along with partial list of listeners. It is responsibility of the caller to
// close the returned non nil listeners whenever required.
//
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if specified socket
//     is not found.
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - [syscall.ENOSYS] is returned on non macOS platforms (including iOS).
func PacketListeners(name string) ([]net.PacketConn, error) {
	files, err := Files(name)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.PacketConn, 0, len(files))
	for _, file := range files {
		l, el := net.FilePacketConn(file)
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

// Deprecated: Use [Listeners].
func TCPListeners(name string) ([]net.Listener, error) {
	return Listeners(name)
}

// Deprecated: Use [PacketListeners].
func UDPListeners(name string) ([]net.PacketConn, error) {
	return PacketListeners(name)
}
