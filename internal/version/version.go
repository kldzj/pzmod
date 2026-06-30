package version

import (
	_ "embed"
)

//go:generate bash version.sh
//go:embed version.txt
var version string

// Get returns the embedded build version (e.g. "v3.0.0"), or "" in dev builds
// where version.txt has not been generated.
func Get() string {
	return version
}

// IsSet reports whether a build version was embedded. Update checks are skipped
// when it is not, so dev builds never reach out to GitHub.
func IsSet() bool {
	return version != ""
}
