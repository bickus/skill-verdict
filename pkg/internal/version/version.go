// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package version

import "runtime/debug"

func String() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "devel"
}
