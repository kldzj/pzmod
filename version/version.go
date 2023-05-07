package version

import (
	_ "embed"
)

//go:embed version.txt
var version string

func Get() string {
	return version
}

func IsSet() bool {
	return version != ""
}
