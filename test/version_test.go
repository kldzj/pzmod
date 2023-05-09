package test

import (
	"strings"
	"testing"

	"github.com/kldzj/pzmod/version"
	"golang.org/x/mod/semver"
)

func TestVersionIsNotEmpty(t *testing.T) {
	if !strings.HasPrefix(version.Get(), "v") {
		t.Errorf("version.Get() does not start with 'v'")
	}

	if strings.Contains(version.Get(), " ") {
		t.Errorf("version.Get() contains a space")
	}

	if strings.Contains(version.Get(), "\n") {
		t.Errorf("version.Get() contains a newline")
	}

	if !semver.IsValid(version.Get()) {
		t.Errorf("version.Get() is not a valid semver version")
	}
}
