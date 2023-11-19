//nolint:goheader // ignore
//
// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build cgo

package testcgo

/*
#include <stdlib.h>
*/
import "C"

func Abs(v int32) int32 {
	return int32(C.abs(C.int(v)))
}
