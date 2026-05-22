// Package version holds build-time identifiers populated via -ldflags at
// `go build` time. Defaults are "dev" / "unknown" so unstamped builds still
// work.
package version

// Version is the semver string (e.g. "0.1.0"). Set via:
//
//	go build -ldflags="-X github.com/dnahilman/region.id/internal/version.Version=0.1.0"
var Version = "dev"

// Commit is the short git SHA. Set via:
//
//	go build -ldflags="-X github.com/dnahilman/region.id/internal/version.Commit=abc1234"
var Commit = "unknown"

// String returns "<version>+<commit>".
func String() string { return Version + "+" + Commit }

// Major returns the leading semver number, used for manifest options-hash
// invalidation. Falls back to "0" for non-semver values.
func Major() string {
	for i, ch := range Version {
		if ch < '0' || ch > '9' {
			if i == 0 {
				return "0"
			}
			return Version[:i]
		}
	}
	return Version
}
