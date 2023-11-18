// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build darwin && !ios

package launchd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"syscall"
	"unsafe"
)

//go:cgo_import_dynamic libc_launch_activate_socket launch_activate_socket "/usr/lib/libSystem.B.dylib"
//nolint:revive,stylecheck // ignore
var libc_launch_activate_socket_trampoline_addr uintptr

//go:cgo_import_dynamic libc_free free "/usr/lib/libSystem.B.dylib"
//nolint:revive,stylecheck // ignore
var libc_free_trampoline_addr uintptr

//go:linkname syscall_syscall syscall.syscall
//nolint:revive,nonamedreturns // ignore
func syscall_syscall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

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
//   - [syscall.EALREADY] is returned if socket is already activated.
//   - [syscall.ENOENT] or [syscall.ESRCH] is returned if specified socket is not
//     found. According to [Apple Launchd Documentation], expected error is
//     [syscall.ENOENT], however implementations return [syscall.ESRCH].
//   - [syscall.ESRCH] is returned if calling process is not manged by launchd.
//   - [syscall.EINVAL] is returned if name contains null characters.
//   - Appropriate unix error code is returned for unexpected errors.
//   - On non macOS platforms (including iOS), this will always return error.
func ListenersWithName(name string) ([]net.Listener, error) {
	var count uint
	var fd uintptr

	cgoName, err := syscall.BytePtrFromString(name)
	if err != nil {
		return nil, fmt.Errorf("launchd: invalid socket name(%s): %w", name, err)
	}

	// Call libc function, launch_activate_socket.
	//
	// int launch_activate_socket(const char *name,  int *_Nonnull *_Nullable fd, size_t *count)
	//
	// Apple's documentation is wrong. It does not mention *fd can be nullable. However,
	// It clearly must be nullable as user is expected to call free on it. Here how it works,
	// You give it a pointer to an uintptr. That uintptr will hold address of fd. Do note that,
	// memory pointed by uintptr is outside of go heap(and not managed by go runtime), and must
	// be de-allocated via free.
	//
	// # Parameters
	//
	//   - name: The name of the socket entry in the service’s Sockets dictionary.
	//   - fds: On return, this parameter is populated with an array of file descriptors.
	//     One socket can have many descriptors associated with it depending on the
	//     characteristics of the network interfaces on the system.
	//     The descriptors in this array are the results of calling getaddrinfo(3) with
	//     the parameters described in launchd.plist. The caller is responsible for
	//     calling free(3) on the returned pointer.
	//   - count: The number of file descriptor entries in the returned array.
	//
	// # Returns
	//
	// On success, 0 is returned. Otherwise, an appropriate POSIX-domain is returned.
	//
	//   - ENOENT, if there was no socket of the specified name owned by the caller.
	//   - ESRCH, if the caller isn’t a process managed by launchd.
	//   - EALREADY, if socket has already been activated by the caller.
	//
	r1, _, e1 := syscall_syscall(
		libc_launch_activate_socket_trampoline_addr,
		uintptr(unsafe.Pointer(cgoName)), // socket name to filter by
		uintptr(unsafe.Pointer(&fd)),     // Pointer to *_Nullable fd
		uintptr(unsafe.Pointer(&count)),  // number of sockets returned
	)

	// Handle syscall error.
	if e1 != 0 {
		return nil, fmt.Errorf("launchd: error calling launch_activate_socket: %w", e1)
	}

	// return code from c-function launch_activate_socket.
	switch r1 {
	case 0:
		if count == 0 {
			// This code is not reachable according do docs, but here for completeness.
			// https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
			return nil, fmt.Errorf("launchd: no sockets found: %w", syscall.ENOENT)
		}

		// This weird unsafe trick is used to silence govet.
		// fd points to memory not managed by go runtime. Also,
		// make a copy of the slice, so that memory backing fd
		// can be de-allocated.
		fdSlice := slices.Clone(
			unsafe.Slice((*int32)(*(*unsafe.Pointer)(unsafe.Pointer(&fd))), int(count)),
		)

		// De-Allocate fd.
		_, _, e1 = syscall_syscall(libc_free_trampoline_addr, fd, 0, 0)
		if e1 != 0 {
			// Technically free returns no error this code is not reachable.
			return nil, fmt.Errorf("launchd: error calling free on *fd: %w", e1)
		}

		// Iterate over file descriptors and create slice of net.Listener.
		listeners := make([]net.Listener, 0, count)
		for _, fd := range fdSlice {
			if fd != 0 {
				file := os.NewFile(uintptr(fd), fmt.Sprintf("launchd-activate-socket-%s", name))
				fl, el := net.FileListener(file)
				if err != nil {
					err = errors.Join(err, el)
					continue
				}
				listeners = append(listeners, fl)
			}
		}

		if err != nil {
			return slices.Clip(listeners), fmt.Errorf("launchd: %w", err)
		}
		return slices.Clip(listeners), nil
	case uintptr(syscall.ENOENT):
		return nil, fmt.Errorf("launchd: no such socket(%s): %w", name, syscall.ENOENT)
	case uintptr(syscall.ESRCH):
		// Weirdly, ESRCH is returned when socket is not present in launchd, not ENOENT
		// as documented. This is most likely a bug in macOS or its documentation.
		//
		// https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
		return nil, fmt.Errorf("launchd: socket/process is not managed by launchd: %w", syscall.ESRCH)
	case uintptr(syscall.EALREADY):
		return nil, fmt.Errorf("launchd: socket(%s) has been already activated: %w", name, syscall.EALREADY)
	default:
		return nil, fmt.Errorf("launchd: unknown error code : %w", syscall.Errno(r1))
	}
}
