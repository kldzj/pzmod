package test

import (
	"strings"
	"testing"

	"github.com/kldzj/pzmod/version"
)

func TestVersionIsNotEmpty(t *testing.T) {
	if version.Get() == "" {
		t.Errorf("version.Get() returned an empty string")
	}

	if !strings.HasPrefix(version.Get(), "v") {
		t.Errorf("version.Get() does not start with 'v'")
	}

	if strings.Contains(version.Get(), " ") {
		t.Errorf("version.Get() contains a space")
	}

	if strings.Contains(version.Get(), "\n") {
		t.Errorf("version.Get() contains a newline")
	}
}
