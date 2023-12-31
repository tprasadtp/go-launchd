// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build darwin && !ios

package launchd

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

const (
	launchDataTypeDict = iota + 1
	launchDataTypeArray
	launchDataTypeFd
	launchDataTypeInteger
	launchDataTypeReal
	launchDataTypeBool
	launchDataTypeString
	launchDataTypeOpaque
	launchDataTypeErrno
	launchDataTypeMachport
)

//go:cgo_import_dynamic libc_launch_data_new_string launch_data_new_string "/usr/lib/libSystem.B.dylib"
//nolint:revive,stylecheck // ignore
var libc_launch_data_new_string_trampoline_addr uintptr

//go:cgo_import_dynamic libc_launch_msg launch_msg "/usr/lib/libSystem.B.dylib"
//nolint:revive,stylecheck // ignore
var libc_launch_msg_trampoline_addr uintptr

//go:cgo_import_dynamic libc_launch_data_get_type launch_data_get_type "/usr/lib/libSystem.B.dylib"
//nolint:revive,stylecheck // ignore
var libc_launch_data_get_type_trampoline_addr uintptr

//go:cgo_import_dynamic libc_launch_data_get_errno launch_data_get_errno "/usr/lib/libSystem.B.dylib"
//nolint:revive,stylecheck // ignore
var libc_launch_data_get_errno_trampoline_addr uintptr

//go:cgo_import_dynamic libc_launch_data_alloc launch_data_alloc "/usr/lib/libSystem.B.dylib"
//nolint:revive,stylecheck // ignore
var libc_launch_data_alloc_trampoline_addr uintptr

//go:cgo_import_dynamic libc_launch_data_free launch_data_free "/usr/lib/libSystem.B.dylib"
//nolint:revive,stylecheck // ignore
var libc_launch_data_free_trampoline_addr uintptr

func isManagedByLaunchd() (bool, error) {
	var errno syscall.Errno

	// Build checkInMsg and pin its memory.
	// This is required as libc might hold references to this go pointer.
	var checkInMsgPinner runtime.Pinner
	checkInMsg, _ := syscall.BytePtrFromString("CheckIn")
	checkInMsgPinner.Pin(&checkInMsg) // This must be **byte
	defer checkInMsgPinner.Unpin()    // unpin in via defer

	var launchMsgString uintptr // points to libc allocated memory.
	launchMsgString, _, errno = syscall_syscall(
		libc_launch_data_new_string_trampoline_addr,
		uintptr(unsafe.Pointer(checkInMsg)),
		0, 0)
	if errno != 0 {
		return false, fmt.Errorf("launchd(libc): error calling launch_data_new_string: %w", errno)
	}

	// Cleanup - launchMsgString
	defer func() {
		_, _, _ = syscall_syscall(
			libc_launch_data_free_trampoline_addr,
			launchMsgString,
			0, 0)
	}()

	if launchMsgString == 0 {
		return false, fmt.Errorf("launchd(libc): launch_data_new_string returned NULL")
	}

	// launch_msg
	var launchMsgResponse uintptr // points to libc allocated memory.
	launchMsgResponse, _, errno = syscall_syscall(
		libc_launch_msg_trampoline_addr,
		launchMsgString,
		0, 0)
	if errno != 0 {
		return false, fmt.Errorf("launchd(libc): error calling launch_msg: %w", errno)
	}
	// Cleanup - launchMsgResponse
	defer func() {
		_, _, _ = syscall_syscall(
			libc_launch_data_free_trampoline_addr,
			launchMsgResponse,
			0, 0)
	}()
	if launchMsgResponse == 0 {
		return false, fmt.Errorf("launchd(libc): launch_msg returned NULL")
	}

	// Check if returned response type is of launchDataTypeErrno.
	var launchMsgResponseType uintptr
	launchMsgResponseType, _, errno = syscall_syscall(
		libc_launch_data_get_type_trampoline_addr,
		launchMsgResponse,
		0, 0)
	if errno != 0 {
		return false, fmt.Errorf("launchd(libc): error calling launch_data_get_type: %w", errno)
	}
	if launchMsgResponseType != launchDataTypeErrno {
		return false, fmt.Errorf("launchd(libc): launch_msg returned unexpected data type: %d", launchMsgResponseType)
	}

	// Get error number from launchMsgResponse
	var launchMsgErrNo uintptr
	launchMsgErrNo, _, errno = syscall_syscall(
		libc_launch_data_get_errno_trampoline_addr,
		launchMsgResponse,
		0, 0)
	if errno != 0 {
		return false, fmt.Errorf("launchd(libc): error calling launch_data_get_errno: %w", errno)
	}

	if launchMsgErrNo == 0 {
		return true, nil
	}
	return false, fmt.Errorf("launchd(libc): launch_msg returned error: %w", syscall.Errno(launchMsgErrNo))
}
