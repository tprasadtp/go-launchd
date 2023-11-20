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

func Abs(v int64) int64 {
	return int64(C.llabs(C.longlong(v)))
}
