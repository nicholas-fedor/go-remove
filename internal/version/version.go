/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package version

import "runtime/debug"

// Unstamped values for the link-time variables.
const (
	// devVersion is the unstamped Version value.
	devVersion = "dev"

	// unknown is the unstamped revision and build time.
	unknown = "unknown"
)

// Link-time build metadata.
var (
	// Version is the application version injected at link time.
	//
	// A value of dev means the binary was not stamped. GetVersion reports the
	// module version in that case and does not assign it back to Version.
	Version = devVersion

	// CommitSHA is the git revision injected at link time.
	CommitSHA = unknown

	// BuildTime is the build timestamp injected at link time.
	BuildTime = unknown
)

// Info is the build metadata for this binary.
type Info struct {
	// Version is the release version, or the module version for a dev build.
	Version string

	// CommitSHA is the source revision stamped at link time.
	CommitSHA string

	// BuildTime is the UTC timestamp stamped at link time.
	BuildTime string
}

// Current returns the build metadata for this binary.
//
// Returns:
//   - Info: build metadata, with Version resolved by GetVersion.
func Current() Info {
	return Info{
		Version:   GetVersion(),
		CommitSHA: CommitSHA,
		BuildTime: BuildTime,
	}
}

// GetVersion returns the current version string.
//
// When Version is dev, the module version from the embedded build info is
// returned and Version is left unchanged.
//
// Returns:
//   - string: the stamped version, or the module version for a dev build.
func GetVersion() string {
	if Version != devVersion {
		return Version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return Version
	}

	return info.Main.Version
}
