// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build darwin && !ios

package launchd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"slices"
	"syscall"
	"unsafe"
)

//go:cgo_import_dynamic libc_launch_activate_socket launch_activate_socket "/usr/lib/libSystem.B.dylib"
//nolint:revive,stylecheck,gochecknoglobals // ignore
var libc_trampoline_launch_activate_socket_addr uintptr

//go:cgo_import_dynamic libc_free free "/usr/lib/libSystem.B.dylib"
//nolint:revive,stylecheck,gochecknoglobals // ignore
var libc_trampoline_free_addr uintptr

// syscall_syscall is implemented in package [runtime] and pushed to [syscall].
//
// Go 1.23 introduces limitations on use of linknames([GH-67401]). However,
// it keeps backward compatibility for [runtime.syscall_syscall] via [ef225d1].
//
// [runtime.syscall_syscall]: https://go.googlesource.com/go/+/ef225d1c57a97af984af114ee52005314530bbe2/src/runtime/sys_darwin.go#23
// [ef225d1]: https://go.googlesource.com/go/+/ef225d1c57a97af984af114ee52005314530bbe2
// [GH-67401]: https://github.com/golang/go/issues/67401
//
//go:linkname syscall_syscall syscall.syscall
//nolint:revive // for linkname
func syscall_syscall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

// listenerFdsWithName returns file descriptors corresponding to the named socket.
func listenerFdsWithName(name string) ([]int32, error) {
	libcName, err := syscall.BytePtrFromString(name)
	if err != nil {
		return nil, fmt.Errorf("launchd: invalid socket name(%s): %w", name, err)
	}

	// Call libc function, launch_activate_socket.
	//
	// int launch_activate_socket(const char *name, int * _Nonnull *fds, size_t *cnt);
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
	// See - https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket

	var fd uintptr // starting address of fds slice (int32)
	var count uint // number of fds

	// Because we are not using syscall.Syscall, but syscall_syscall directly,
	// which does not use "go:uintptrkeepalive" directive. Pin go pointers
	// passed to libc code.

	var pinner runtime.Pinner
	pinner.Pin(&fd)
	pinner.Pin(&count)
	pinner.Pin(&libcName)
	defer pinner.Unpin()

	// Use syscall_syscall as it does some magic to avoid errors.
	// Using syscall.Syscall will result in invalid args and panic.
	// Though syscall.syscall_syscall is not exported, it is extensively
	// used by the [golang.org/x/sys/unix] package and thus is fairly
	// reliable.
	//
	// https://github.com/golang/go/issues/65355 (check if syscall.syscall_syscall is moved here)
	// https://github.com/golang/go/issues/67401 (resolved)
	// https://github.com/golang/go/issues/51087
	r1, _, e1 := syscall_syscall(
		libc_trampoline_launch_activate_socket_addr,
		uintptr(unsafe.Pointer(libcName)), // socket name to filter by
		uintptr(unsafe.Pointer(&fd)),      // Pointer to *fds
		uintptr(unsafe.Pointer(&count)),   // number of sockets
	)

	if e1 != 0 {
		return nil, fmt.Errorf("launchd: error calling launch_activate_socket: %w", e1)
	}

	// return code from c-function launch_activate_socket.
	switch r1 {
	case 0:
		if count == 0 {
			// This code is not reachable, according do docs, but here for completeness.
			return nil, fmt.Errorf("launchd: no sockets found: %w", syscall.ENOENT)
		}

		// - As *fd points to memory not managed by go runtime, make a copy
		//   of the slice after building it.
		// - Unsafe trick is used to silence govet.
		fdSlice := slices.Clone(
			unsafe.Slice((*int32)(*(*unsafe.Pointer)(unsafe.Pointer(&fd))), int(count)),
		)

		// de-allocate *fd.
		_, _, e1 = syscall_syscall(libc_trampoline_free_addr, fd, 0, 0)
		if e1 != 0 {
			return nil, fmt.Errorf("launchd: error calling free on *fd: %w", e1)
		}

		// Return file descriptors.
		return fdSlice, nil
	case uintptr(syscall.ENOENT):
		return nil, fmt.Errorf("launchd: no such socket(%s): %w", name, syscall.ENOENT)
	case uintptr(syscall.ESRCH):
		// Weirdly, ESRCH is returned when the socket is not present in launchd,
		// not ENOENT as documented. This is most likely a bug in macOS or its
		// documentation.
		//
		// https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
		return nil, fmt.Errorf("launchd: socket/process is not managed by launchd: %w", syscall.ESRCH)
	case uintptr(syscall.EALREADY):
		return nil, fmt.Errorf("launchd: socket(%s) has been already activated: %w", name, syscall.EALREADY)
	default:
		return nil, fmt.Errorf("launchd: unknown error code : %w", syscall.Errno(r1))
	}
}

// Os specific implementation of [Files].
func files(name string) ([]*os.File, error) {
	fdSlice, err := listenerFdsWithName(name)
	if err != nil {
		return nil, err
	}
	files := make([]*os.File, 0, len(fdSlice))
	for _, fd := range fdSlice {
		if fd != 0 {
			files = append(files, os.NewFile(uintptr(fd),
				fmt.Sprintf("%s-io.github.tprasadtp.go-launchd.socket", name)))
		}
	}
	return slices.Clip(files), nil
}

// Os specific implementation of [Listeners].
func listeners(name string) ([]net.Listener, error) {
	files, err := Files(name)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(files))
	for _, file := range files {
		stype, stypeErr := syscall.GetsockoptInt(int(file.Fd()), syscall.SOL_SOCKET, syscall.SO_TYPE)
		if stypeErr != nil {
			err = errors.Join(err, os.NewSyscallError("getsockopt", stypeErr))
			continue
		}

		if stype != syscall.SOCK_STREAM {
			err = errors.Join(err, fmt.Errorf("%s: %w", name, syscall.ESOCKTNOSUPPORT))
			continue
		}

		l, el := net.FileListener(file)
		if el != nil {
			err = errors.Join(err, el)
		} else {
			listeners = append(listeners, l)
		}
	}

	if err != nil {
		return slices.Clip(listeners), fmt.Errorf("launchd: error building listeners: %w", err)
	}
	return slices.Clip(listeners), nil
}

// Os specific implementation of [PacketListeners].
func packetListeners(name string) ([]net.PacketConn, error) {
	files, err := Files(name)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.PacketConn, 0, len(files))
	for _, file := range files {
		stype, stypeErr := syscall.GetsockoptInt(int(file.Fd()), syscall.SOL_SOCKET, syscall.SO_TYPE)
		if stypeErr != nil {
			err = errors.Join(err, os.NewSyscallError("getsockopt", stypeErr))
			continue
		}

		if stype != syscall.SOCK_DGRAM {
			err = errors.Join(err, fmt.Errorf("%s: %w", name, syscall.ESOCKTNOSUPPORT))
			continue
		}

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
