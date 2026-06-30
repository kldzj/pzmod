package domain

import "testing"

func TestParseModRef(t *testing.T) {
	cases := []struct{ in, ws, id string }{
		{"tsarslib", "", "tsarslib"},
		{`\tsarslib`, "", "tsarslib"},
		{`2392709985\tsarslib`, "2392709985", "tsarslib"},
		{`  \ Spaced `, "", "Spaced"},
		{"", "", ""},
		{`\`, "", ""},
	}
	for _, c := range cases {
		r := ParseModRef(c.in)
		if r.Workshop != c.ws || r.ID != c.id {
			t.Errorf("ParseModRef(%q) = {ws:%q id:%q}; want {ws:%q id:%q}", c.in, r.Workshop, r.ID, c.ws, c.id)
		}
		if r.Raw != c.in {
			t.Errorf("ParseModRef(%q).Raw = %q; want %q", c.in, r.Raw, c.in)
		}
	}
}

func TestFormatModRef(t *testing.T) {
	cases := []struct {
		ws, id string
		expl   bool
		want   string
	}{
		{"2392709985", "tsarslib", true, `2392709985\tsarslib`},
		{"", "tsarslib", true, `\tsarslib`},
		{"2392709985", "tsarslib", false, "tsarslib"},
		{"", "", true, ""},
	}
	for _, c := range cases {
		if got := FormatModRef(c.ws, c.id, c.expl); got != c.want {
			t.Errorf("FormatModRef(%q,%q,%v) = %q; want %q", c.ws, c.id, c.expl, got, c.want)
		}
	}
}

func TestDedupeMods(t *testing.T) {
	got := DedupeMods([]string{`\X`, "X", `W1\X`, `W2\X`, `\Y`, "", `\`})
	want := []string{`\X`, `W1\X`, `W2\X`, `\Y`}
	if len(got) != len(want) {
		t.Fatalf("DedupeMods = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DedupeMods = %v; want %v", got, want)
		}
	}
}
