// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build !darwin || ios

package launchd

import (
	"fmt"
)

func listenersFdsWithName(_ string) ([]int32, error) {
	return nil, fmt.Errorf("launchd: only supported on macOS")
}
