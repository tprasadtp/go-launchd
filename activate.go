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
// In case of error building listeners, an appropriate error is returned,
// along with partial list of listeners. It is responsibility of the caller to
// close the returned listeners whenever required.
//
// Multiple [net.Listener] may be returned when listening on both IPv4 and
// IPv6. Ensure that your server correctly handles listening on multiple
// [net.Listener].
//
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if specified socket is not
//     found. According to [Apple Launchd Documentation], expected error is
//     [syscall.ENOENT], however implementations return [syscall.ESRCH].
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - [syscall.ENOSYS] is returned on non macOS platforms (including iOS).
//
// [Apple Launchd Documentation]: https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
func TCPListeners(name string) ([]net.Listener, error) {
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

// UDPListeners returns slice of [net.PacketConn] for specified UDP
// socket name, as  mentioned in launchd plist file.
//
// In case of error building listeners, an appropriate error is returned,
// along with partial list of listeners. It is responsibility of the caller to
// close the returned listeners whenever required.
//
// Multiple [net.PacketConn] may be returned when listening on both IPv4 and
// IPv6. Ensure that your server correctly handles listening on multiple
// [net.Listener].
//
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if specified socket is not
//     found. According to [Apple Launchd Documentation], expected error is
//     [syscall.ENOENT], however implementations return [syscall.ESRCH].
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - [syscall.ENOSYS] is returned on non macOS platforms (including iOS).
//
// [Apple Launchd Documentation]: https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
func UDPListeners(name string) ([]net.PacketConn, error) {
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

// Files returns slice of [*os.File] backed by file descriptor for socket name,
// as  mentioned in launchd plist file.
//
// In case of error building listeners, an appropriate error is returned,
// along with partial list of listeners. It is responsibility of the caller to
// close the returned listeners whenever required.
//
// Multiple files may be returned when listening on both IPv4 and
// IPv6. Ensure that your server correctly handles listening on multiple
// [net.Listener].
//
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if specified socket is not
//     found. According to [Apple Launchd Documentation], expected error is
//     [syscall.ENOENT], however implementations return [syscall.ESRCH].
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - [syscall.ENOSYS] is returned on non macOS platforms (including iOS).
//
// [Apple Launchd Documentation]: https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
func Files(name string) ([]*os.File, error) {
	fdSlice, err := listenerFdsWithName(name)
	if err != nil {
		return nil, err
	}
	files := make([]*os.File, 0, len(fdSlice))
	for _, fd := range fdSlice {
		if fd != 0 {
			files = append(files, os.NewFile(uintptr(fd),
				fmt.Sprintf("launchd-socket-%s", name)))
		}
	}
	return slices.Clip(files), nil
}
