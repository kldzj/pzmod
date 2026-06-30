package serverconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const fixture = "testdata/servertest.ini"

// Ported from the v2 TestIniIntegrity: parsing then rendering the real fixture
// must be byte-for-byte identical.
func TestFixtureRoundTrip(t *testing.T) {
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	c := FromBytes(fixture, data)
	if got := c.Bytes(); string(got) != string(data) {
		t.Errorf("fixture did not round-trip byte-for-byte")
	}
}

// Ported from v2 TestLoadIni.
func TestLoadValues(t *testing.T) {
	c, err := Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Name(); got != "pzmod" {
		t.Errorf("Name = %q; want pzmod", got)
	}
	if got := c.MaxPlayers(); got != "32" {
		t.Errorf("MaxPlayers = %q; want 32", got)
	}
	if got := c.Mods(); len(got) != 0 {
		t.Errorf("Mods = %v; want empty", got)
	}
	if got := c.WorkshopItems(); !reflect.DeepEqual(got, []string{"2849247394"}) {
		t.Errorf("WorkshopItems = %v; want [2849247394]", got)
	}
}

// Map values contain commas that are part of the name, not separators.
func TestMapsKeepCommas(t *testing.T) {
	c, _ := Load(fixture)
	if got := c.Maps(); !reflect.DeepEqual(got, []string{"Muldraugh, KY"}) {
		t.Errorf("Maps = %v; want [\"Muldraugh, KY\"]", got)
	}
}

func TestSplitFixedSeparators(t *testing.T) {
	c := FromBytes("x.ini", []byte("Mods=a;b , c;;d ; b\n"))
	want := []string{"a", "b", "c", "d"}
	if got := c.Mods(); !reflect.DeepEqual(got, want) {
		t.Errorf("Mods = %v; want %v", got, want)
	}
}

// Ported from v2 TestSaveIni.
func TestSaveRoundTrip(t *testing.T) {
	c, _ := Load(fixture)
	out := filepath.Join(t.TempDir(), "out.ini")
	if err := c.SaveTo(out); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(out)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.String() != c.String() {
		t.Error("save/reload differs from original")
	}
}

// The key new guarantee: editing mod lists must not disturb any other bytes
// (comments, blank lines, EOL, unrelated keys).
func TestEditModsPreservesEverythingElse(t *testing.T) {
	data, _ := os.ReadFile(fixture)
	c := FromBytes(fixture, data)

	c.SetWorkshopItems([]string{"2849247394", "123456"})
	c.SetMods([]string{"ModA", "ModB"})

	got := c.String()

	// Only the two edited lines should differ from the original.
	origLines := splitKeepLine(string(data))
	gotLines := splitKeepLine(got)
	if len(origLines) != len(gotLines) {
		t.Fatalf("line count changed: %d -> %d", len(origLines), len(gotLines))
	}
	var diffs []int
	for i := range origLines {
		if origLines[i] != gotLines[i] {
			diffs = append(diffs, i)
		}
	}
	if len(diffs) != 2 {
		t.Fatalf("expected exactly 2 changed lines, got %d (%v)", len(diffs), diffs)
	}
	if c.GetOr(KeyWorkshop, "") != "2849247394;123456" {
		t.Errorf("WorkshopItems value = %q", c.GetOr(KeyWorkshop, ""))
	}
	if c.GetOr(KeyMods, "") != "ModA;ModB" {
		t.Errorf("Mods value = %q", c.GetOr(KeyMods, ""))
	}
}

func TestB42ModsRoundTripAndDedupe(t *testing.T) {
	in := "Mods=\\tsarslib;2392709985\\Containers;PlainMod\nWorkshopItems=2392709985\nMap=Muldraugh, KY\n"
	c := FromBytes("mem.ini", []byte(in))

	// Unmodified: byte-for-byte identical (raw tokens preserved, incl. backslashes).
	if got := string(c.Bytes()); got != in {
		t.Errorf("round-trip changed bytes:\n got %q\nwant %q", got, in)
	}
	// Mods() returns the raw tokens, not stripped.
	want := []string{"\\tsarslib", "2392709985\\Containers", "PlainMod"}
	if got := c.Mods(); !reflect.DeepEqual(got, want) {
		t.Errorf("Mods() = %v; want %v", got, want)
	}
	// SetMods dedupes by mod ref: \X and X collapse; W1\X and W2\X are kept.
	c.SetMods([]string{"\\X", "X", "W1\\X", "W2\\X"})
	if got, _ := c.Get("Mods"); got != "\\X;W1\\X;W2\\X" {
		t.Errorf("SetMods dedupe = %q; want %q", got, "\\X;W1\\X;W2\\X")
	}
}

func splitKeepLine(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		cur += string(r)
		if r == '\n' {
			out = append(out, cur)
			cur = ""
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
