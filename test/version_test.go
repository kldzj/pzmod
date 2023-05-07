package test

import (
	"os"
	"strings"
	"testing"

	"github.com/kldzj/pzmod/version"
)

func TestVersionIsNotEmpty(t *testing.T) {
	ensureCiCdEnv(t)
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

func ensureCiCdEnv(t *testing.T) {
	if os.Getenv("CI") != "true" {
		t.Skip("Skipping test because CI is not set")
	}
}
