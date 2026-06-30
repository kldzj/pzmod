// Package pathutil holds small filesystem path helpers shared by the CLI and TUI.
package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// Expand resolves a leading "~" to the user's home directory and returns an
// absolute, cleaned path. It is a no-op on error so callers always get a usable
// string back.
func Expand(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				p = home
			} else {
				p = filepath.Join(home, p[2:])
			}
		}
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// Abbreviate replaces a leading home-directory prefix with "~" for display. It
// never errors: if the home directory can't be determined or p is not under it,
// p is returned unchanged. This keeps shown paths short and avoids leaking the
// OS username in screenshots.
func Abbreviate(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(os.PathSeparator)) {
		return "~" + p[len(home):]
	}
	return p
}

// FileExists reports whether p exists and is a regular file (not a directory).
func FileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// DirExists reports whether p exists and is a directory.
func DirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}
