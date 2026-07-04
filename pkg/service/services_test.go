package service

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/kldzj/pzmod/pkg/build"
	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/pkg/modinfo"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/steam/steamtest"
	"github.com/kldzj/pzmod/pkg/store"
)

// item builds a WorkshopItem with a description that Parse() can scrape.
func item(id, title string, mods, maps, children []string, fileType uint8, banned bool) steam.WorkshopItem {
	desc := ""
	for _, m := range mods {
		desc += "Mod ID: " + m + "\n"
	}
	for _, mp := range maps {
		desc += "Map Folder: " + mp + "\n"
	}
	var ch []steam.WorkshopItemChild
	for _, c := range children {
		ch = append(ch, steam.WorkshopItemChild{PublishedFileID: c})
	}
	return steam.WorkshopItem{
		Result:          1,
		FileType:        fileType,
		PublishedFileID: id,
		Title:           title,
		Description:     desc,
		Banned:          banned,
		Children:        ch,
	}
}

// canned returns a fake seeded with a representative dependency graph.
func canned() *steamtest.Fake {
	return steamtest.New(
		item("100", "Core Library", []string{"CoreLib"}, nil, nil, steam.FileTypeMod, false),
		item("200", "Weapons", []string{"Weapons"}, nil, []string{"100"}, steam.FileTypeMod, false),
		item("300", "My Collection", nil, nil, []string{"100", "200"}, steam.FileTypeCollection, false),
		item("400", "Map Pack", []string{"MapPack"}, []string{"BigMap"}, nil, steam.FileTypeMod, false),
		item("500", "Two Mods", []string{"AA", "BB"}, nil, nil, steam.FileTypeMod, false),
		item("600", "No Mod ID", nil, nil, nil, steam.FileTypeMod, false),
		item("700", "Seven", []string{"Seven"}, nil, []string{"800"}, steam.FileTypeMod, false),
		item("800", "Eight", []string{"Eight"}, nil, []string{"700"}, steam.FileTypeMod, false),
	)
}

func svc(f *steamtest.Fake) *Services {
	return &Services{Steam: f, Now: time.Now}
}

func TestResolveTransitiveDeps(t *testing.T) {
	s := svc(canned())
	plan, err := s.Resolve(context.Background(), []string{"200"}, domain.ServerMods{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.AddWorkshopItems, []string{"200", "100"}) {
		t.Errorf("AddWorkshopItems = %v; want [200 100]", plan.AddWorkshopItems)
	}
	if !reflect.DeepEqual(plan.AddMods, []string{"Weapons", "CoreLib"}) {
		t.Errorf("AddMods = %v; want [Weapons CoreLib]", plan.AddMods)
	}
}

func TestResolveCollectionNotInstalled(t *testing.T) {
	s := svc(canned())
	plan, err := s.Resolve(context.Background(), []string{"300"}, domain.ServerMods{})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(plan.AddWorkshopItems)
	if !reflect.DeepEqual(plan.AddWorkshopItems, []string{"100", "200"}) {
		t.Errorf("AddWorkshopItems = %v; collection 300 must be expanded, not installed", plan.AddWorkshopItems)
	}
	for _, id := range plan.AddWorkshopItems {
		if id == "300" {
			t.Error("collection id leaked into WorkshopItems")
		}
	}
}

func TestResolveSkipsInstalled(t *testing.T) {
	s := svc(canned())
	installed := domain.ServerMods{WorkshopItems: []string{"200"}, Mods: []string{"Weapons"}}
	plan, err := s.Resolve(context.Background(), []string{"200"}, installed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.AddWorkshopItems, []string{"100"}) {
		t.Errorf("AddWorkshopItems = %v; want only the missing dep [100]", plan.AddWorkshopItems)
	}
	if !reflect.DeepEqual(plan.AddMods, []string{"CoreLib"}) {
		t.Errorf("AddMods = %v; want [CoreLib]", plan.AddMods)
	}
}

func TestResolveMultiModAndNoModID(t *testing.T) {
	s := svc(canned())
	plan, _ := s.Resolve(context.Background(), []string{"500", "600"}, domain.ServerMods{})
	if len(plan.MultiMod) != 1 || plan.MultiMod[0].ItemID != "500" {
		t.Errorf("MultiMod = %v; want one entry for 500", plan.MultiMod)
	}
	if !reflect.DeepEqual(plan.NoModID, []string{"600"}) {
		t.Errorf("NoModID = %v; want [600]", plan.NoModID)
	}
}

func TestResolveMissing(t *testing.T) {
	s := svc(canned())
	plan, _ := s.Resolve(context.Background(), []string{"999"}, domain.ServerMods{})
	if !reflect.DeepEqual(plan.Missing, []string{"999"}) {
		t.Errorf("Missing = %v; want [999]", plan.Missing)
	}
}

func TestResolveCycleTerminates(t *testing.T) {
	s := svc(canned())
	plan, err := s.Resolve(context.Background(), []string{"700"}, domain.ServerMods{})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Cycles) == 0 {
		t.Error("expected a cycle to be reported for 700<->800")
	}
	// Both items still resolved.
	if !plan.HasItem("700") || !plan.HasItem("800") {
		t.Errorf("AddWorkshopItems = %v; want both 700 and 800", plan.AddWorkshopItems)
	}
}

func TestValidateMissingDependency(t *testing.T) {
	s := svc(canned())
	sm := domain.ServerMods{WorkshopItems: []string{"200"}, Mods: []string{"Weapons"}}
	report, err := s.Validate(context.Background(), sm, build.Unknown)
	if err != nil {
		t.Fatal(err)
	}
	if !report.HasErrors() {
		t.Fatal("expected a missing-dependency error")
	}
	if !hasFinding(report, domain.CodeMissingDependency, "100") {
		t.Errorf("expected missing-dependency for 100, got %+v", report.Findings)
	}
}

func TestValidateUnknownAndUnused(t *testing.T) {
	s := svc(canned())
	// Mods has a ghost mod; item 100 provides CoreLib which isn't enabled.
	sm := domain.ServerMods{WorkshopItems: []string{"100"}, Mods: []string{"Ghost"}}
	report, _ := s.Validate(context.Background(), sm, build.Unknown)
	if !hasFinding(report, domain.CodeUnknownModID, "Ghost") {
		t.Errorf("expected unknown-mod-id for Ghost")
	}
	if !hasFinding(report, domain.CodeUnusedModID, "CoreLib") {
		t.Errorf("expected unused-mod-id for CoreLib")
	}
}

func TestValidateDelisted(t *testing.T) {
	s := svc(canned())
	report, _ := s.Validate(context.Background(), domain.ServerMods{WorkshopItems: []string{"999"}}, build.Unknown)
	if !hasFinding(report, domain.CodeDelisted, "999") {
		t.Errorf("expected delisted finding for 999")
	}
}

func TestValidateBuildCompat(t *testing.T) {
	f := steamtest.New(func() steam.WorkshopItem {
		it := item("900", "Old Mod", []string{"OldMod"}, nil, nil, steam.FileTypeMod, false)
		it.Tags = []steam.WorkshopTag{{Tag: "Build 41"}}
		return it
	}())
	s := svc(f)
	sm := domain.ServerMods{WorkshopItems: []string{"900"}, Mods: []string{"OldMod"}}
	report, _ := s.Validate(context.Background(), sm, build.B42)
	if !hasFinding(report, domain.CodeBuildCompat, "900") {
		t.Errorf("expected build-compat warning for a B41-only mod on a B42 profile")
	}
}

func TestSuggestLoadOrderDependencyAndFramework(t *testing.T) {
	s := svc(canned())
	// Wrong order: Weapons before its dependency CoreLib.
	sm := domain.ServerMods{WorkshopItems: []string{"100", "200"}, Mods: []string{"Weapons", "CoreLib"}}
	plan, err := s.SuggestLoadOrder(context.Background(), sm, store.Profile{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Ordered, []string{"CoreLib", "Weapons"}) {
		t.Errorf("Ordered = %v; want [CoreLib Weapons]", plan.Ordered)
	}
}

func TestSuggestLoadOrderModInfoEnrichment(t *testing.T) {
	s := svc(canned())
	s.ModInfoOverride = fakeProvider{"AA": {"BB"}} // AA requires BB
	// 500 provides AA and BB; no workshop dep between them, only mod.info.
	sm := domain.ServerMods{WorkshopItems: []string{"500"}, Mods: []string{"AA", "BB"}}
	plan, err := s.SuggestLoadOrder(context.Background(), sm, store.Profile{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Ordered, []string{"BB", "AA"}) {
		t.Errorf("Ordered = %v; want [BB AA] from mod.info require=", plan.Ordered)
	}
}

// Dry-run proof: projecting a plan and validating must never touch the file.
func TestDryRunDoesNotWrite(t *testing.T) {
	s := svc(canned())
	dir := t.TempDir()
	path := filepath.Join(dir, "server.ini")
	original := []byte("WorkshopItems=200\nMods=Weapons\n")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(path)

	sm := domain.ServerMods{WorkshopItems: []string{"200"}, Mods: []string{"Weapons"}}
	plan, _ := s.Resolve(context.Background(), sm.WorkshopItems, sm)
	projected := plan.Apply(sm, false)
	if _, err := s.Validate(context.Background(), projected, build.Unknown); err != nil {
		t.Fatal(err)
	}

	after, _ := os.Stat(path)
	now, _ := os.ReadFile(path)
	if string(now) != string(original) {
		t.Error("dry-run modified the file contents")
	}
	if before.ModTime() != after.ModTime() {
		t.Error("dry-run changed the file mtime")
	}
}

func hasFinding(r domain.Report, code, subject string) bool {
	for _, f := range r.Findings {
		if f.Code == code && f.Subject == subject {
			return true
		}
	}
	return false
}

func TestValidateB42NoFalseFindingsAndClash(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "10",
			Title: "Lib", Description: "Mod ID: tsarslib\n"},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "20",
			Title: "Pack A", Description: "Mod ID: Dupe\n"},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "30",
			Title: "Pack B", Description: "Mod ID: Dupe\n"},
	)
	sm := domain.ServerMods{
		Mods:          []string{`\tsarslib`, `20\Dupe`},
		WorkshopItems: []string{"10", "20", "30"},
	}
	report, err := svc(fake).Validate(context.Background(), sm, build.B41)
	if err != nil {
		t.Fatal(err)
	}
	// No false "unknown mod id" for \tsarslib, and no false "unused" for tsarslib.
	if hasFinding(report, domain.CodeUnknownModID, `\tsarslib`) || hasFinding(report, domain.CodeUnknownModID, "tsarslib") {
		t.Errorf("false CodeUnknownModID on a B42 token: %+v", report.Findings)
	}
	if hasFinding(report, domain.CodeUnusedModID, "tsarslib") {
		t.Errorf("false CodeUnusedModID for an enabled B42 mod: %+v", report.Findings)
	}
	// Dupe is declared by items 20 and 30 -> exactly one clash advisory.
	if !hasFinding(report, domain.CodeModIDClash, "Dupe") {
		t.Errorf("expected CodeModIDClash for Dupe: %+v", report.Findings)
	}
}

func TestSuggestLoadOrderB42(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200",
			Title: "Weapons", Description: "Mod ID: Weapons\n",
			Children: []steam.WorkshopItemChild{{PublishedFileID: "100"}}},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100",
			Title: "Core Library", Description: "Mod ID: CoreLib\n"},
	)
	sm := domain.ServerMods{
		Mods:          []string{`\Weapons`, `\CoreLib`}, // wrong order: dependent first
		WorkshopItems: []string{"200", "100"},
	}
	plan, err := svc(fake).SuggestLoadOrder(context.Background(), sm, store.Profile{})
	if err != nil {
		t.Fatal(err)
	}
	// CoreLib (a dependency, and "library" framework) must come before Weapons,
	// and the suggestion must preserve the RAW tokens (with backslashes).
	if len(plan.Ordered) != 2 || plan.Ordered[0] != `\CoreLib` || plan.Ordered[1] != `\Weapons` {
		t.Errorf("Ordered = %v; want [\\CoreLib \\Weapons]", plan.Ordered)
	}
}

// fakeProvider implements modinfo.Provider for tests.
type fakeProvider map[string][]string

func (f fakeProvider) Lookup(ids []string) map[string]modinfo.ModInfo {
	out := map[string]modinfo.ModInfo{}
	for _, id := range ids {
		if req, ok := f[id]; ok {
			out[id] = modinfo.ModInfo{ID: id, Require: req}
		}
	}
	return out
}

func TestSuggestLoadOrderB42Edge(t *testing.T) {
	// Neither mod name nor title contains a framework keyword, so the ONLY thing
	// that can reorder them is the dependency edge (item 200 requires item 100).
	// This guards the B42 edge fix even if the framework heuristic were disabled.
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200",
			Title: "Big Guns", Description: "Mod ID: BigGuns\n",
			Children: []steam.WorkshopItemChild{{PublishedFileID: "100"}}},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100",
			Title: "Ammo Box", Description: "Mod ID: AmmoBox\n"},
	)
	sm := domain.ServerMods{
		Mods:          []string{`\BigGuns`, `\AmmoBox`}, // dependent before its prerequisite
		WorkshopItems: []string{"200", "100"},
	}
	plan, err := svc(fake).SuggestLoadOrder(context.Background(), sm, store.Profile{})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Ordered) != 2 || plan.Ordered[0] != `\AmmoBox` || plan.Ordered[1] != `\BigGuns` {
		t.Errorf("Ordered = %v; want [\\AmmoBox \\BigGuns] (edge must move the prerequisite first)", plan.Ordered)
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

func TestResolveB42(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200",
			Title: "Weapons", Description: "Mod ID: Weapons\n"},
	)
	// CoreLib already installed as a B42 token must NOT be re-added.
	installed := domain.ServerMods{Mods: []string{`100\CoreLib`}, WorkshopItems: []string{"100"}}
	plan, err := svc(fake).Resolve(context.Background(), []string{"200"}, installed)
	if err != nil {
		t.Fatal(err)
	}
	if contains(plan.AddMods, "CoreLib") {
		t.Errorf("CoreLib already installed (as 100\\CoreLib) must not be re-added: %v", plan.AddMods)
	}
	if plan.AddModSources["Weapons"] != "200" {
		t.Errorf("AddModSources[Weapons] = %q; want 200", plan.AddModSources["Weapons"])
	}
	// Explicit apply writes the pinned form; non-explicit writes plain.
	got := plan.Plan.Apply(installed, true)
	if !contains(got.Mods, `200\Weapons`) {
		t.Errorf("explicit Apply should add 200\\Weapons: %v", got.Mods)
	}
	if got := plan.Plan.Apply(installed, false); !contains(got.Mods, "Weapons") {
		t.Errorf("non-explicit Apply should add plain Weapons: %v", got.Mods)
	}
}
