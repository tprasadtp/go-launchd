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

// TCPListeners returns slice of [net.Listener] for specified TCP socket,
// as mentioned in launchd plist file.
//
// This does not make use of cgo and makes calls to using libc directly.
// This is obviously not supported on iOS.
//
// In case of error building listeners, an appropriate error is returned,
// along with partial list of listeners. It is responsibility of the caller to
// close the returned listeners whenever required.
//
// Multiple file descriptors are returned when listening on both IPv4 and
// IPv6. Ensure that your server correctly handles listening on multiple
// [net.Listener].
//
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if specified socket is not
//     found. According to [Apple Launchd Documentation], expected error is
//     [syscall.ENOENT], however implementations return [syscall.ESRCH].
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - Appropriate unix error code is returned for unexpected errors.
//   - On non macOS platforms (including iOS), this will always return error.
//
// [Apple Launchd Documentation]: https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
func TCPListeners(name string) ([]net.Listener, error) {
	fdSlice, err := listenerFdsWithName(name)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(fdSlice))
	for _, fd := range fdSlice {
		if fd != 0 {
			file := os.NewFile(uintptr(fd), fmt.Sprintf("launchd-activate-tcp-socket-%s", name))
			fl, el := net.FileListener(file)
			if el != nil {
				err = errors.Join(err, el)
			} else {
				listeners = append(listeners, fl)
			}
		}
	}

	if err != nil {
		return slices.Clip(listeners), fmt.Errorf("launchd: %w", err)
	}
	return slices.Clip(listeners), nil
}

// UDPListeners returns slice of [net.PacketConn] for specified UDP
// socket name, as  mentioned in launchd plist file.
//
// This does not make use of cgo and makes calls to using libc directly.
// This is obviously not supported on iOS.
//
// In case of error building listeners, an appropriate error is returned,
// along with partial list of listeners. It is responsibility of the caller to
// close the returned listeners whenever required.
//
// Multiple file descriptors are returned when listening on both IPv4 and
// IPv6. Ensure that your server correctly handles listening on multiple
// [net.Listener].
//
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if specified socket is not
//     found. According to [Apple Launchd Documentation], expected error is
//     [syscall.ENOENT], however implementations return [syscall.ESRCH].
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - Appropriate unix error code is returned for unexpected errors.
//   - On non macOS platforms (including iOS), this will always return error.
//
// [Apple Launchd Documentation]: https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
func UDPListeners(name string) ([]net.PacketConn, error) {
	fdSlice, err := listenerFdsWithName(name)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.PacketConn, 0, len(fdSlice))
	for _, fd := range fdSlice {
		if fd != 0 {
			file := os.NewFile(uintptr(fd), fmt.Sprintf("launchd-activate-udp-socket-%s", name))
			fl, el := net.FilePacketConn(file)
			if el != nil {
				err = errors.Join(err, el)
			} else {
				listeners = append(listeners, fl)
			}
		}
	}

	if err != nil {
		return slices.Clip(listeners), fmt.Errorf("launchd: %w", err)
	}
	return slices.Clip(listeners), nil
}
