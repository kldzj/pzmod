package modinfo

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeModInfo(t *testing.T, root, workshopID, modName, content string) {
	t.Helper()
	dir := filepath.Join(root, workshopID, "mods", modName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mod.info"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDiskProviderLookup(t *testing.T) {
	root := t.TempDir()
	writeModInfo(t, root, "111", "CoreLib", "name=CoreLib\nid=CoreLib\n")
	writeModInfo(t, root, "222", "Weapons", "name=Weapons\nrequire=CoreLib,OtherLib\n")

	p := NewProvider(root)
	got := p.Lookup([]string{"Weapons", "CoreLib", "Absent"})

	if len(got) != 2 {
		t.Fatalf("got %d infos; want 2 (%v)", len(got), got)
	}
	if !reflect.DeepEqual(got["Weapons"].Require, []string{"CoreLib", "OtherLib"}) {
		t.Errorf("Weapons require = %v", got["Weapons"].Require)
	}
	// id falls back to name when id= is absent.
	writeModInfo(t, root, "333", "NameOnly", "name=NameOnly\n")
	if mi := NewProvider(root).Lookup([]string{"NameOnly"}); mi["NameOnly"].ID != "NameOnly" {
		t.Errorf("name-only id = %q; want NameOnly", mi["NameOnly"].ID)
	}
}

func TestNopProviderWhenNoRoot(t *testing.T) {
	if got := NewProvider("").Lookup([]string{"x"}); len(got) != 0 {
		t.Errorf("empty-root provider should return nothing, got %v", got)
	}
}

func TestMissingRootIsSafe(t *testing.T) {
	if got := NewProvider("/nonexistent/path/xyz").Lookup([]string{"x"}); len(got) != 0 {
		t.Errorf("missing root should be safe, got %v", got)
	}
}
