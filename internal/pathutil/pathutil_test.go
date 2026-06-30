package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := Expand("~/foo"); got != filepath.Join(home, "foo") {
		t.Errorf("Expand(~/foo) = %q; want %q", got, filepath.Join(home, "foo"))
	}
	if got := Expand("~"); got != home {
		t.Errorf("Expand(~) = %q; want %q", got, home)
	}
	// ~user is NOT expanded (only ~ and ~/).
	if got := Expand("~bob/x"); got == filepath.Join(home, "bob/x") {
		t.Errorf("Expand(~bob/x) should not expand to home")
	}
}

func TestAbbreviate(t *testing.T) {
	home, _ := os.UserHomeDir()
	if home == "" {
		t.Skip("no home dir")
	}
	under := filepath.Join(home, "Zomboid", "Server", "servertest.ini")
	if got := Abbreviate(under); got != "~/Zomboid/Server/servertest.ini" {
		t.Errorf("Abbreviate(%q) = %q; want ~/Zomboid/Server/servertest.ini", under, got)
	}
	if got := Abbreviate(home); got != "~" {
		t.Errorf("Abbreviate(home) = %q; want ~", got)
	}
	// A path not under home is returned unchanged.
	if got := Abbreviate("/etc/passwd"); got != "/etc/passwd" {
		t.Errorf("Abbreviate(/etc/passwd) = %q; want unchanged", got)
	}
	// A sibling that merely shares the home prefix string is not abbreviated.
	if got := Abbreviate(home + "extra"); got != home+"extra" {
		t.Errorf("Abbreviate(%q) = %q; want unchanged", home+"extra", got)
	}
}

func TestExistsHelpers(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.ini")
	os.WriteFile(f, []byte("x"), 0644)
	if !FileExists(f) || FileExists(dir) {
		t.Error("FileExists wrong for file/dir")
	}
	if !DirExists(dir) || DirExists(f) {
		t.Error("DirExists wrong for dir/file")
	}
}
