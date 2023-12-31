// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build !darwin || ios

package launchd

func isManagedByLaunchd() (bool, error) {
	return false, nil
}
