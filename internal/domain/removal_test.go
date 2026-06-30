package domain

import (
	"reflect"
	"testing"
)

func TestPlanRemoval(t *testing.T) {
	current := ServerMods{
		WorkshopItems: []string{"1", "2"},
		Mods:          []string{"Shared", "OnlyA", "OnlyB", "Manual"},
		Maps:          []string{"MapA", "Base"},
	}
	decl := map[string]ModDecl{
		"1": {Mods: []string{"Shared", "OnlyA"}, Maps: []string{"MapA"}},
		"2": {Mods: []string{"Shared", "OnlyB"}, Maps: nil},
	}

	got := PlanRemoval("1", decl, current)
	want := RemovalPlan{Item: "1", Mods: []string{"OnlyA"}, Maps: []string{"MapA"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PlanRemoval=%+v want %+v", got, want)
	}

	// Removing an item we have no declarations for removes only the item.
	got2 := PlanRemoval("99", map[string]ModDecl{}, current)
	if len(got2.Mods) != 0 || len(got2.Maps) != 0 || got2.Item != "99" {
		t.Fatalf("unexpected plan for unknown item: %+v", got2)
	}

	// Item "3" is declared but not installed - must not protect "OnlyA".
	current3 := ServerMods{
		WorkshopItems: []string{"1"}, // "3" absent
		Mods:          []string{"Shared", "OnlyA"},
		Maps:          []string{"MapA"},
	}
	decl3 := map[string]ModDecl{
		"1": {Mods: []string{"Shared", "OnlyA"}, Maps: []string{"MapA"}},
		"3": {Mods: []string{"OnlyA"}, Maps: nil}, // declared but not installed
	}
	got3 := PlanRemoval("1", decl3, current3)
	want3 := RemovalPlan{Item: "1", Mods: []string{"Shared", "OnlyA"}, Maps: []string{"MapA"}}
	if !reflect.DeepEqual(got3, want3) {
		t.Fatalf("installed-set guard: PlanRemoval=%+v want %+v", got3, want3)
	}
}

func TestRemovalPlanApply(t *testing.T) {
	// Create a scenario: item "1" and "2" share "Shared", "1" uniquely owns "OnlyA" and map "MapA", plus manual mod.
	current := ServerMods{
		WorkshopItems: []string{"1", "2"},
		Mods:          []string{"Shared", "OnlyA", "Manual"},
		Maps:          []string{"MapA", "Base"},
	}
	decl := map[string]ModDecl{
		"1": {Mods: []string{"Shared", "OnlyA"}, Maps: []string{"MapA"}},
		"2": {Mods: []string{"Shared"}, Maps: nil},
	}

	// Plan removal of item "1".
	plan := PlanRemoval("1", decl, current)

	// Apply the plan and verify result.
	result := plan.Apply(current)

	// Item "1" should be removed.
	if result.HasItem("1") {
		t.Fatalf("item '1' should be removed, got=%+v", result)
	}

	// Item "2" should remain.
	if !result.HasItem("2") {
		t.Fatalf("item '2' should remain, got=%+v", result)
	}

	// "OnlyA" (uniquely owned by "1") should be removed.
	if result.HasMod("OnlyA") {
		t.Fatalf("mod 'OnlyA' should be removed, got=%+v", result)
	}

	// "Shared" (shared with "2") should remain.
	if !result.HasMod("Shared") {
		t.Fatalf("mod 'Shared' should remain, got=%+v", result)
	}

	// "Manual" (manually added) should remain.
	if !result.HasMod("Manual") {
		t.Fatalf("mod 'Manual' should remain, got=%+v", result)
	}

	// "MapA" (uniquely owned by "1") should be removed.
	if result.HasMap("MapA") {
		t.Fatalf("map 'MapA' should be removed, got=%+v", result)
	}

	// "Base" (manually added) should remain.
	if !result.HasMap("Base") {
		t.Fatalf("map 'Base' should remain, got=%+v", result)
	}
}
