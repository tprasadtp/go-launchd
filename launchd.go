// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package launchd

// IsManagedByLaunchd returns true if process is managed by launchd.
// Returned bool is only valid if error is nil.
func IsManagedByLaunchd() (bool, error) {
	return isManagedByLaunchd()
}
