package steam

import (
	"reflect"
	"testing"
)

func TestParseScopesToOwnWorkshopID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		desc     string
		wantMods []string
		wantMaps []string
	}{
		{
			name: "advertised footer is ignored (real 3042138819 case)",
			id:   "3042138819",
			desc: "The new multiplayer true music jukebox is now available!\n" +
				"True Music Jukebox\n" +
				"https://steamcommunity.com/sharedfiles/filedetails/?id=3118990099\n" +
				"Workshop ID: 3118990099\n" +
				"Mod ID: TrueMusicJukebox\n" +
				"\n" +
				"Workshop ID: 3042138819\n" +
				"Mod ID: FunctionalAppliances2\n",
			wantMods: []string{"FunctionalAppliances2"},
			wantMaps: []string{},
		},
		{
			name: "several mod ids under the item's own workshop id are all kept",
			id:   "555",
			desc: "A bundle.\n" +
				"Workshop ID: 555\n" +
				"Mod ID: ModA\n" +
				"Mod ID: ModB\n",
			wantMods: []string{"ModA", "ModB"},
			wantMaps: []string{},
		},
		{
			name: "reversed footer (Mod ID before Workshop ID, blank between) is kept (real 3654929003 case)",
			id:   "3654929003",
			desc: "Other batch action work:\n" +
				"https://steamcommunity.com/sharedfiles/filedetails/?id=3584890848 batch recipe action\n" +
				"https://steamcommunity.com/sharedfiles/filedetails/?id=3660401764 pick everything on floor\n" +
				"\n" +
				"Mod ID: vac_mod_b42_4\n" +
				"\n" +
				"Workshop ID: 3654929003\n",
			wantMods: []string{"vac_mod_b42_4"},
			wantMaps: []string{},
		},
		{
			name: "other mods advertised only as URLs do not cause drops",
			id:   "100",
			desc: "Check out my other mods:\n" +
				"https://steamcommunity.com/sharedfiles/filedetails/?id=999\n" +
				"Mod ID: MyMod\n" +
				"Workshop ID: 100\n",
			wantMods: []string{"MyMod"},
			wantMaps: []string{},
		},
		{
			name: "a separated foreign footer block is dropped, own block kept",
			id:   "100",
			desc: "Workshop ID: 999\n" +
				"Mod ID: ForeignMod\n" +
				"Map Folder: ForeignMap\n" +
				"\n" +
				"\n" +
				"Workshop ID: 100\n" +
				"Mod ID: MyMod\n" +
				"Map Folder: MyMap\n",
			wantMods: []string{"MyMod"},
			wantMaps: []string{"MyMap"},
		},
		{
			name: "no workshop id line falls back to keeping everything",
			id:   "42",
			desc: "A simple mod.\n" +
				"Mod ID: Simple\n" +
				"Map Folder: SimpleMap\n",
			wantMods: []string{"Simple"},
			wantMaps: []string{"SimpleMap"},
		},
		{
			name: "casing variants are tolerated",
			id:   "88",
			desc: "WorkshopID: 88\n" +
				"ModID: CasedMod\n",
			wantMods: []string{"CasedMod"},
			wantMaps: []string{},
		},
		{
			name: "only a foreign workshop id present keeps everything (own id absent)",
			id:   "200",
			desc: "Check out my other mod:\n" +
				"Workshop ID: 999\n" +
				"Mod ID: OtherMod\n" +
				"Mod ID: BareMod\n",
			wantMods: []string{"OtherMod", "BareMod"},
			wantMaps: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := WorkshopItem{PublishedFileID: tc.id, Description: tc.desc}
			got := w.Parse()
			if !reflect.DeepEqual(got.Mods, tc.wantMods) {
				t.Errorf("mods = %#v, want %#v", got.Mods, tc.wantMods)
			}
			if !reflect.DeepEqual(got.Maps, tc.wantMaps) {
				t.Errorf("maps = %#v, want %#v", got.Maps, tc.wantMaps)
			}
		})
	}
}
