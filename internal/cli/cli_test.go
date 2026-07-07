package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kldzj/pzmod/pkg/serverconfig"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/steam/steamtest"
	"github.com/kldzj/pzmod/pkg/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	s, err := store.New(store.WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// run executes one command with a fresh root (clean flag state) against st.
func run(t *testing.T, st *store.Store, args ...string) (string, error) {
	t.Helper()
	root := NewRootCommand(st, "test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func useFakeSteam(t *testing.T, f steam.API) {
	t.Helper()
	old := steamFactory
	steamFactory = func(string) steam.API { return f }
	t.Cleanup(func() { steamFactory = old })
}

func writeINI(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "server.ini")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGetSetRoundTrip(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "PublicName=old\nMaxPlayers=16\n")

	out, err := run(t, st, "get", "name", "--file", ini)
	if err != nil || strings.TrimSpace(out) != "old" {
		t.Fatalf("get name = %q, %v; want old", out, err)
	}

	if _, err := run(t, st, "set", "name", "new name", "--file", ini); err != nil {
		t.Fatal(err)
	}
	out, _ = run(t, st, "get", "name", "--file", ini)
	if strings.TrimSpace(out) != "new name" {
		t.Errorf("after set, get name = %q; want 'new name'", out)
	}

	// Byte fidelity: only the PublicName line changed.
	data, _ := os.ReadFile(ini)
	if string(data) != "PublicName=new name\nMaxPlayers=16\n" {
		t.Errorf("file content = %q", data)
	}
}

func TestGetList(t *testing.T) {
	st := testStore(t)
	out, _ := run(t, st, "get", "list")
	for _, want := range []string{"name", "desc", "public", "password", "slots"} {
		if !strings.Contains(out, want) {
			t.Errorf("get list missing %q in %q", want, out)
		}
	}
}

func TestProfileLifecycle(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "PublicName=x\n")

	if _, err := run(t, st, "profile", "add", "--name", "Alpha", "--file", ini, "--build", "b42"); err != nil {
		t.Fatal(err)
	}
	out, _ := run(t, st, "profile", "list")
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "[B42]") {
		t.Errorf("profile list = %q", out)
	}
	// Default profile resolves without --file.
	out, _ = run(t, st, "get", "name")
	if strings.TrimSpace(out) != "x" {
		t.Errorf("default-profile get = %q; want x", out)
	}
}

func cannedFake() *steamtest.Fake {
	mk := func(id, title string, mods, children []string, ft uint8) steam.WorkshopItem {
		desc := ""
		for _, m := range mods {
			desc += "Mod ID: " + m + "\n"
		}
		var ch []steam.WorkshopItemChild
		for _, c := range children {
			ch = append(ch, steam.WorkshopItemChild{PublishedFileID: c})
		}
		return steam.WorkshopItem{Result: 1, FileType: ft, PublishedFileID: id, Title: title, Description: desc, Children: ch}
	}
	return steamtest.New(
		mk("100", "Core Library", []string{"CoreLib"}, nil, steam.FileTypeMod),
		mk("200", "Weapons", []string{"Weapons"}, []string{"100"}, steam.FileTypeMod),
	)
}

func TestValidateExitsNonZeroOnError(t *testing.T) {
	st := testStore(t)
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")
	useFakeSteam(t, cannedFake())
	// 200 requires 100, which is not installed -> missing-dependency error.
	ini := writeINI(t, "WorkshopItems=200\nMods=Weapons\n")

	out, err := run(t, st, "validate", "--file", ini)
	if err == nil {
		t.Errorf("validate should exit non-zero on errors; out=%q", out)
	}
	if !strings.Contains(out, "missing dependency") {
		t.Errorf("validate output missing dependency text: %q", out)
	}
}

func TestModsAddResolveDeps(t *testing.T) {
	st := testStore(t)
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")
	useFakeSteam(t, cannedFake())
	ini := writeINI(t, "WorkshopItems=\nMods=\n")

	if _, err := run(t, st, "mods", "add", "200", "--resolve-deps", "--file", ini); err != nil {
		t.Fatal(err)
	}
	cfg, _ := serverconfig.Load(ini)
	items := cfg.WorkshopItems()
	if len(items) != 2 || !contains(items, "100") || !contains(items, "200") {
		t.Errorf("WorkshopItems = %v; want 100 and 200 (dep resolved)", items)
	}
	if mods := cfg.Mods(); !contains(mods, "CoreLib") || !contains(mods, "Weapons") {
		t.Errorf("Mods = %v; want CoreLib and Weapons", mods)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func TestModsAddDryRunDoesNotWrite(t *testing.T) {
	st := testStore(t)
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")
	useFakeSteam(t, cannedFake())
	const original = "WorkshopItems=\nMods=\n"
	ini := writeINI(t, original)

	out, err := run(t, st, "mods", "add", "200", "--resolve-deps", "--dry-run", "--file", ini, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got resolveJSON
	if uerr := json.Unmarshal([]byte(out), &got); uerr != nil {
		t.Fatalf("unmarshal %q: %v", out, uerr)
	}
	if !got.DryRun {
		t.Errorf("dryRun = false; want true")
	}
	if !contains(got.AddWorkshopItems, "100") || !contains(got.AddWorkshopItems, "200") {
		t.Errorf("addWorkshopItems = %v; want 100 and 200", got.AddWorkshopItems)
	}
	data, _ := os.ReadFile(ini)
	if string(data) != original {
		t.Errorf("file changed under --dry-run: %q", data)
	}
}

func TestSetDryRunDoesNotWrite(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "PublicName=old\nMaxPlayers=16\n")

	out, err := run(t, st, "set", "name", "new", "--dry-run", "--file", ini, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got setPreviewJSON
	if uerr := json.Unmarshal([]byte(out), &got); uerr != nil {
		t.Fatalf("unmarshal %q: %v", out, uerr)
	}
	if got.Old != "old" || got.New != "new" || !got.DryRun {
		t.Errorf("preview = %+v", got)
	}
	data, _ := os.ReadFile(ini)
	if string(data) != "PublicName=old\nMaxPlayers=16\n" {
		t.Errorf("file changed under --dry-run: %q", data)
	}
}

func TestDoctorClean(t *testing.T) {
	st := testStore(t)
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")
	useFakeSteam(t, cannedFake())
	// 100 is installed with its mod id; no missing deps -> clean.
	ini := writeINI(t, "WorkshopItems=100\nMods=CoreLib\n")

	out, err := run(t, st, "doctor", "--file", ini, "--json")
	if err != nil {
		t.Fatalf("doctor should pass on a clean config; err=%v out=%q", err, out)
	}
	var got doctorJSON
	if uerr := json.Unmarshal([]byte(out), &got); uerr != nil {
		t.Fatalf("unmarshal %q: %v", out, uerr)
	}
	if !got.OK {
		t.Errorf("ok = false; want true. checks=%+v", got.Checks)
	}
}

func TestDoctorValidationError(t *testing.T) {
	st := testStore(t)
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")
	useFakeSteam(t, cannedFake())
	// 200 requires 100, which is not installed -> validation error.
	ini := writeINI(t, "WorkshopItems=200\nMods=Weapons\n")

	out, err := run(t, st, "doctor", "--file", ini, "--json")
	if err == nil {
		t.Errorf("doctor should exit non-zero on a validation error; out=%q", out)
	}
	var got doctorJSON
	if uerr := json.Unmarshal([]byte(out), &got); uerr != nil {
		t.Fatalf("unmarshal %q: %v", out, uerr)
	}
	if got.OK {
		t.Errorf("ok = true; want false")
	}
}

func TestDoctorOfflineSkipsValidation(t *testing.T) {
	st := testStore(t)
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")
	useFakeSteam(t, cannedFake())
	ini := writeINI(t, "WorkshopItems=200\nMods=Weapons\n")

	out, err := run(t, st, "doctor", "--file", ini, "--offline", "--json")
	if err != nil {
		t.Errorf("offline doctor should not fail on validation; err=%v", err)
	}
	if !strings.Contains(out, "\"validation\"") || !strings.Contains(out, "\"skip\"") {
		t.Errorf("expected validation skip in %q", out)
	}
}

func TestDoctorBadFile(t *testing.T) {
	st := testStore(t)
	out, err := run(t, st, "doctor", "--file", "/no/such/server.ini", "--json")
	if err == nil {
		t.Errorf("doctor should exit non-zero when the config cannot load; out=%q", out)
	}
}

func TestModsRemoveDryRunDoesNotWrite(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "WorkshopItems=100;200\nMods=CoreLib;Weapons\n")

	out, err := run(t, st, "mods", "remove", "100", "CoreLib", "--dry-run", "--file", ini, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got removePreviewJSON
	if uerr := json.Unmarshal([]byte(out), &got); uerr != nil {
		t.Fatalf("unmarshal %q: %v", out, uerr)
	}
	if !got.DryRun || !contains(got.RemovedItems, "100") || !contains(got.RemovedMods, "CoreLib") {
		t.Errorf("preview = %+v", got)
	}
	data, _ := os.ReadFile(ini)
	if string(data) != "WorkshopItems=100;200\nMods=CoreLib;Weapons\n" {
		t.Errorf("file changed under --dry-run: %q", data)
	}
}

func TestCompleteProfiles(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "PublicName=x\n")
	if _, err := run(t, st, "profile", "add", "--name", "Alpha", "--file", ini); err != nil {
		t.Fatal(err)
	}
	got, _ := completeProfiles(st)(nil, nil, "")
	if !contains(got, "alpha") {
		t.Errorf("profiles completion = %v; want alpha", got)
	}
}

func TestCompleteInstalledIDs(t *testing.T) {
	st := testStore(t)
	ini := writeINI(t, "WorkshopItems=100;200\nMods=CoreLib\nMap=Springfield\n")
	cmd := newModsRemoveCmd(st)
	if err := cmd.Flags().Set("file", ini); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.ValidArgsFunction(cmd, nil, "")
	for _, want := range []string{"100", "200", "CoreLib", "Springfield"} {
		if !contains(got, want) {
			t.Errorf("installed completion missing %q in %v", want, got)
		}
	}
}

func TestCompleteConfigKeys(t *testing.T) {
	got, _ := completeConfigKeys(nil, nil, "")
	if !contains(got, "name") {
		t.Errorf("config keys completion = %v; want name", got)
	}
}
