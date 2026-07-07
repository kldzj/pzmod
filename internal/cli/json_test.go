package cli

import (
	"encoding/json"
	"testing"
)

func TestModsListJSON(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "WorkshopItems=100;200\nMods=CoreLib;Weapons\nMap=Springfield;Muldraugh, KY\n")

	out, err := run(t, st, "mods", "list", "--file", ini, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got modsListJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal %q: %v", out, err)
	}
	if len(got.Mods) != 2 || got.Mods[0] != "CoreLib" {
		t.Errorf("mods = %v", got.Mods)
	}
	if len(got.WorkshopItems) != 2 || !contains(got.WorkshopItems, "100") {
		t.Errorf("workshopItems = %v", got.WorkshopItems)
	}
	// Map splits on ';' only, so "Muldraugh, KY" stays one entry.
	if len(got.Maps) != 2 || got.Maps[1] != "Muldraugh, KY" {
		t.Errorf("maps = %v", got.Maps)
	}
}

func TestModsListJSONEmptyArrays(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "WorkshopItems=\nMods=\n")

	out, _ := run(t, st, "mods", "list", "--file", ini, "--json")
	// nil slices must serialize as [] for machine consumers, not null.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("unmarshal %q: %v", out, err)
	}
	for _, k := range []string{"mods", "workshopItems", "maps"} {
		if string(raw[k]) != "[]" {
			t.Errorf("%s = %s; want []", k, raw[k])
		}
	}
}

func TestValidateJSONError(t *testing.T) {
	st := testStore(t)
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")
	useFakeSteam(t, cannedFake())
	// 200 requires 100, which is not installed -> missing-dependency error.
	ini := writeINI(t, "WorkshopItems=200\nMods=Weapons\n")

	out, err := run(t, st, "validate", "--file", ini, "--json")
	if err == nil {
		t.Errorf("validate --json should still exit non-zero on errors; out=%q", out)
	}
	var got validateJSON
	if uerr := json.Unmarshal([]byte(out), &got); uerr != nil {
		t.Fatalf("unmarshal %q: %v", out, uerr)
	}
	if got.OK {
		t.Errorf("ok = true; want false")
	}
	if got.Summary.Errors < 1 {
		t.Errorf("summary.errors = %d; want >= 1", got.Summary.Errors)
	}
	found := false
	for _, f := range got.Findings {
		if f.Code == "missing-dependency" {
			found = true
			if f.Severity != "ERROR" {
				t.Errorf("severity = %q; want ERROR", f.Severity)
			}
		}
	}
	if !found {
		t.Errorf("no missing-dependency finding in %+v", got.Findings)
	}
}

func TestSearchJSON(t *testing.T) {
	st := testStore(t)
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")
	useFakeSteam(t, cannedFake())

	out, err := run(t, st, "search", "Weapons", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got searchJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal %q: %v", out, err)
	}
	if got.Total != 1 || len(got.Items) != 1 {
		t.Fatalf("search = %+v; want 1 item", got)
	}
	if got.Items[0].ID != "200" || got.Items[0].Title != "Weapons" {
		t.Errorf("item = %+v", got.Items[0])
	}
}

func TestProfileListJSON(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "PublicName=x\n")
	if _, err := run(t, st, "profile", "add", "--name", "Alpha", "--file", ini, "--build", "b42"); err != nil {
		t.Fatal(err)
	}

	out, err := run(t, st, "profile", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got profileListJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal %q: %v", out, err)
	}
	if len(got.Profiles) != 1 {
		t.Fatalf("profiles = %+v", got.Profiles)
	}
	p := got.Profiles[0]
	if p.ID != "alpha" || p.Build != "b42" || !p.Default {
		t.Errorf("profile = %+v", p)
	}
	if got.DefaultID != "alpha" {
		t.Errorf("defaultId = %q; want alpha", got.DefaultID)
	}
}

func TestBackupListJSON(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "PublicName=x\n")

	// Empty case normalizes to [].
	out, _ := run(t, st, "backup", "list", "--file", ini, "--json")
	var empty map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &empty); err != nil {
		t.Fatalf("unmarshal %q: %v", out, err)
	}
	if string(empty["backups"]) != "[]" {
		t.Errorf("empty backups = %s; want []", empty["backups"])
	}

	// After a snapshot, the list carries one entry.
	if _, err := run(t, st, "backup", "snapshot", "--file", ini, "--note", "hi", "--json"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, st, "backup", "list", "--file", ini, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got backupListJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal %q: %v", out, err)
	}
	if len(got.Backups) != 1 || got.Backups[0].Kind != "manual" || got.Backups[0].Note != "hi" {
		t.Errorf("backups = %+v", got.Backups)
	}
}

func TestGetJSON(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "PublicName=Hello World\n")

	out, err := run(t, st, "get", "name", "--file", ini, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got getJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal %q: %v", out, err)
	}
	if got.Key != "name" || got.Value != "Hello World" {
		t.Errorf("get = %+v", got)
	}
}
