package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFilterMatch(t *testing.T) {
	cases := []struct {
		q      string
		fields []string
		want   bool
	}{
		{"", []string{"anything"}, true},
		{"hydro", []string{"Hydrocraft"}, true},
		{"HYDRO", []string{"hydrocraft mod"}, true},
		{"craft", []string{"Hydrocraft"}, true},
		{"zzz", []string{"Hydrocraft", "Weapons"}, false},
		{"weap", []string{"Hydrocraft", "Weapons"}, true},
	}
	for _, c := range cases {
		if got := filterMatch(c.q, c.fields...); got != c.want {
			t.Errorf("filterMatch(%q, %v) = %v; want %v", c.q, c.fields, got, c.want)
		}
	}
}

func TestFilterStateTransitions(t *testing.T) {
	var f filterState
	if f.has() || f.active {
		t.Fatal("zero value should be inactive/empty")
	}
	f.start()
	if !f.active {
		t.Fatal("start() should activate")
	}
	f.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	f.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	if f.query != "ab" {
		t.Fatalf("query = %q; want ab", f.query)
	}
	f.handleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if f.query != "a" {
		t.Fatalf("after backspace query = %q; want a", f.query)
	}
	if !f.handleKey(tea.KeyMsg{Type: tea.KeyEnter}) {
		t.Fatal("enter should be consumed")
	}
	if f.active || !f.has() || f.query != "a" {
		t.Fatalf("after enter: active=%v query=%q; want inactive, query=a", f.active, f.query)
	}
	f.start()
	if !f.handleKey(tea.KeyMsg{Type: tea.KeyEsc}) {
		t.Fatal("esc should be consumed")
	}
	if f.active || f.has() {
		t.Fatalf("after esc: active=%v query=%q; want cleared", f.active, f.query)
	}
	if f.handleKey(tea.KeyMsg{Type: tea.KeyUp}) {
		t.Fatal("up arrow should NOT be consumed by handleKey")
	}
}
