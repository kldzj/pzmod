package version

import (
	_ "embed"
)

//go:generate bash version.sh
//go:embed version.txt
var version string

func Get() string {
	return version
}

func IsSet() bool {
	return version != ""
}
