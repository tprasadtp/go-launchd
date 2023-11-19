// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package launchd

import "net"

// ListenersWithName returns slice of [net.Listener] for specified socket name,
// as  mentioned in launchd plist file.
//
// This does not make use of cgo and makes calls to using system library directly.
// This is obviously not supported on ios.
//
// It is responsibility of the caller to close the returned listeners whenever
// required. They are not closed even when there is an error. In case of error
// building listeners, an appropriate error is returned along with partial list
// of listeners.
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
func ListenersWithName(name string) ([]net.Listener, error) {
	return listenersWithName(name)
}
