// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build !cgo

package testcgo

// This is dummy function which always returns incorrect result.
func Abs(v int32) int32 {
	return -1
}
