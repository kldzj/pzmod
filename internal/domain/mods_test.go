package domain

import (
	"reflect"
	"testing"
)

func TestDedupe(t *testing.T) {
	got := Dedupe([]string{"a", "", "b", "a", "c", "b", ""})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Dedupe = %v; want %v", got, want)
	}
}

func TestAddRemove(t *testing.T) {
	s := ServerMods{Mods: []string{"a"}}
	s2 := s.AddMod("b").AddMod("a") // "a" already present
	if !reflect.DeepEqual(s2.Mods, []string{"a", "b"}) {
		t.Errorf("AddMod = %v", s2.Mods)
	}
	s3 := s2.RemoveMod("a")
	if !reflect.DeepEqual(s3.Mods, []string{"b"}) {
		t.Errorf("RemoveMod = %v", s3.Mods)
	}
}

func TestCloneNoAlias(t *testing.T) {
	s := ServerMods{Mods: []string{"a", "b"}}
	c := s.Clone()
	c.Mods[0] = "x"
	if s.Mods[0] != "a" {
		t.Error("Clone aliases the original slice")
	}
}

func TestAddMapBaseLast(t *testing.T) {
	cases := []struct {
		name     string
		start    ServerMods
		add      string
		wantMaps []string
	}{
		{
			name:     "custom after base goes before base",
			start:    ServerMods{Maps: []string{"Muldraugh, KY"}},
			add:      "Basements",
			wantMaps: []string{"Basements", "Muldraugh, KY"},
		},
		{
			name:     "empty list adds custom",
			start:    ServerMods{},
			add:      "Basements",
			wantMaps: []string{"Basements"},
		},
		{
			name:     "existing custom and base, add another custom before base",
			start:    ServerMods{Maps: []string{"Basements", "Muldraugh, KY"}},
			add:      "Riverside",
			wantMaps: []string{"Basements", "Riverside", "Muldraugh, KY"},
		},
		{
			name:     "adding a base map appends at end",
			start:    ServerMods{Maps: []string{"Basements"}},
			add:      "West Point, KY",
			wantMaps: []string{"Basements", "West Point, KY"},
		},
		{
			name:     "duplicate custom leaves maps unchanged",
			start:    ServerMods{Maps: []string{"Basements", "Muldraugh, KY"}},
			add:      "Basements",
			wantMaps: []string{"Basements", "Muldraugh, KY"},
		},
		{
			name:     "multiple bases, custom inserted before first base",
			start:    ServerMods{Maps: []string{"Muldraugh, KY", "Riverside, KY"}},
			add:      "Custom",
			wantMaps: []string{"Custom", "Muldraugh, KY", "Riverside, KY"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.start.AddMap(tc.add)
			if !reflect.DeepEqual(got.Maps, tc.wantMaps) {
				t.Errorf("AddMap(%q): got %v; want %v", tc.add, got.Maps, tc.wantMaps)
			}
		})
	}
}

func TestAddIgnoresEmptyAndDup(t *testing.T) {
	s := ServerMods{}.AddItem("1").AddItem("").AddItem("1").AddMap("Muldraugh, KY")
	if !reflect.DeepEqual(s.WorkshopItems, []string{"1"}) {
		t.Errorf("WorkshopItems = %v", s.WorkshopItems)
	}
	if !reflect.DeepEqual(s.Maps, []string{"Muldraugh, KY"}) {
		t.Errorf("Maps = %v", s.Maps)
	}
}

func TestServerModsRefAware(t *testing.T) {
	sm := ServerMods{Mods: []string{`\tsarslib`, `2392709985\Containers`, "PlainMod"}}
	for _, id := range []string{"tsarslib", "Containers", "PlainMod"} {
		if !sm.HasMod(id) {
			t.Errorf("HasMod(%q) = false; want true", id)
		}
	}
	if sm.HasMod("Nope") {
		t.Error("HasMod(Nope) = true; want false")
	}
	// AddMod is a no-op when the ID is already enabled in another form.
	if got := sm.AddMod("tsarslib"); len(got.Mods) != len(sm.Mods) {
		t.Errorf("AddMod(tsarslib) should be a no-op; got %v", got.Mods)
	}
	// AddMod appends a new token verbatim.
	if got := sm.AddMod(`\NewMod`); !contains(got.Mods, `\NewMod`) {
		t.Errorf("AddMod(\\NewMod) not added: %v", got.Mods)
	}
	// RemoveMod drops the raw token by logical ID.
	rm := sm.RemoveMod("tsarslib")
	if rm.HasMod("tsarslib") || contains(rm.Mods, `\tsarslib`) {
		t.Errorf("RemoveMod(tsarslib) should drop \\tsarslib: %v", rm.Mods)
	}
}
