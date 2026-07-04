package ini

import (
	"strings"
	"testing"
)

// roundTrip is the core contract: an unmodified document must render
// byte-for-byte identical to its source.
func TestRoundTripByteExact(t *testing.T) {
	cases := map[string]string{
		"empty":               "",
		"single no newline":   "Key=Value",
		"single newline":      "Key=Value\n",
		"trailing blank":      "Key=Value\n\n",
		"multiple blanks":     "A=1\n\n\n\nB=2\n",
		"comment then key":    "# a comment\nKey=Value\n",
		"inline hash value":   "Welcome=Hello #1 player\n",
		"leading blank":       "\n\nKey=Value\n",
		"crlf":                "A=1\r\nB=2\r\n",
		"cr only":             "A=1\rB=2\r",
		"mixed endings":       "A=1\r\nB=2\nC=3\rD=4",
		"duplicate keys":      "K=1\nK=2\nK=3\n",
		"spaces around":       "  Key = Value  \n",
		"non entry line":      "this is not an entry\nKey=Value\n",
		"value with equals":   "Key=a=b=c\n",
		"semicolon comment":   "; not treated as comment here\nKey=Value\n",
		"only blanks":         "\n\n\n",
		"no trailing newline": "A=1\nB=2",
	}

	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			got := Parse([]byte(in)).String()
			if got != in {
				t.Errorf("round-trip mismatch\n in: %q\nout: %q", in, got)
			}
		})
	}
}

func TestGet(t *testing.T) {
	d := Parse([]byte("# comment\nPublicName=pzmod\nMaxPlayers = 32 \nMods=\nDup=1\nDup=2\n"))

	if v, ok := d.Get("PublicName"); !ok || v != "pzmod" {
		t.Errorf("PublicName = %q, %v; want pzmod, true", v, ok)
	}
	if v, ok := d.Get("MaxPlayers"); !ok || v != "32" {
		t.Errorf("MaxPlayers = %q, %v; want 32, true (trimmed)", v, ok)
	}
	if v, ok := d.Get("Mods"); !ok || v != "" {
		t.Errorf("Mods = %q, %v; want empty, true", v, ok)
	}
	if _, ok := d.Get("Missing"); ok {
		t.Error("Missing should not be found")
	}
	// Duplicate keys: Get returns the first occurrence.
	if v, _ := d.Get("Dup"); v != "1" {
		t.Errorf("Dup = %q; want 1 (first occurrence)", v)
	}
}

func TestInlineCommentStripping(t *testing.T) {
	cases := map[string]string{
		"Name=Comment Heavy        # inline comment\n": "Comment Heavy",
		"Name=value # note\n":                          "value",
		"Name=value\t# tabbed note\n":                  "value",
		"Name=#hexish\n":                               "#hexish", // no preceding space: kept
		"Name=a;b;c # mods\n":                          "a;b;c",
		"Name=plain\n":                                 "plain",
		"Name= # only a comment\n":                     "",
	}
	for in, want := range cases {
		if got, _ := Parse([]byte(in)).Get("Name"); got != want {
			t.Errorf("Get(Name) for %q = %q; want %q", in, got, want)
		}
	}
}

func TestSetPreservesInlineComment(t *testing.T) {
	d := Parse([]byte("Name=old value   # keep me\n"))
	d.Set("Name", "new value")
	if got, want := d.String(), "Name=new value # keep me\n"; got != want {
		t.Errorf("set with inline comment\n got: %q\nwant: %q", got, want)
	}
	// And the round trip stays stable after MarkSaved.
	d.MarkSaved()
	if d.HasUnsavedChanges() {
		t.Error("MarkSaved should clear dirty")
	}
	if got := d.String(); got != "Name=new value # keep me\n" {
		t.Errorf("post-MarkSaved render = %q", got)
	}
}

func TestSetExistingPreservesLayout(t *testing.T) {
	in := "# header comment\nPublicName=old\n\n# trailing\nMaxPlayers=16\n"
	d := Parse([]byte(in))
	d.Set("PublicName", "new")

	want := "# header comment\nPublicName=new\n\n# trailing\nMaxPlayers=16\n"
	if got := d.String(); got != want {
		t.Errorf("set existing\n got: %q\nwant: %q", got, want)
	}
	if !d.HasUnsavedChanges() {
		t.Error("expected HasUnsavedChanges after Set")
	}
}

func TestSetNewKeyAppends(t *testing.T) {
	d := Parse([]byte("A=1\n"))
	d.Set("B", "2")
	if got, want := d.String(), "A=1\nB=2\n"; got != want {
		t.Errorf("append new key\n got: %q\nwant: %q", got, want)
	}
}

func TestSetNewKeyAfterNoTrailingNewline(t *testing.T) {
	d := Parse([]byte("A=1")) // no trailing newline
	d.Set("B", "2")
	if got, want := d.String(), "A=1\nB=2\n"; got != want {
		t.Errorf("append after no-newline\n got: %q\nwant: %q", got, want)
	}
}

func TestSetPreservesCRLF(t *testing.T) {
	d := Parse([]byte("A=1\r\nB=2\r\n"))
	d.Set("A", "9")
	if got, want := d.String(), "A=9\r\nB=2\r\n"; got != want {
		t.Errorf("set with crlf\n got: %q\nwant: %q", got, want)
	}
}

func TestHasUnsavedChangesAndMarkSaved(t *testing.T) {
	d := Parse([]byte("A=1\n"))
	if d.HasUnsavedChanges() {
		t.Error("fresh parse should have no unsaved changes")
	}
	d.Set("A", "2")
	if !d.HasUnsavedChanges() {
		t.Error("expected unsaved changes after Set")
	}
	d.MarkSaved()
	if d.HasUnsavedChanges() {
		t.Error("MarkSaved should clear unsaved changes")
	}
	if v, _ := d.Get("A"); v != "2" {
		t.Errorf("value after MarkSaved = %q; want 2", v)
	}
}

func FuzzRoundTrip(f *testing.F) {
	seeds := []string{
		"", "Key=Value", "A=1\nB=2\n", "\r\n\r\n", "# c\nk=v\r\n",
		"K=1\nK=2", "no equals here", "x=a=b\n\n",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		got := Parse([]byte(in)).String()
		if got != in {
			t.Errorf("round-trip not byte-exact\n in: %q\nout: %q", in, got)
		}
	})
}

// Sanity check that EOL detection drives appended lines.
func TestAppendUsesDetectedEOL(t *testing.T) {
	d := Parse([]byte("A=1\r\n"))
	d.Set("B", "2")
	if !strings.HasSuffix(d.String(), "B=2\r\n") {
		t.Errorf("appended line should use detected CRLF, got %q", d.String())
	}
}
