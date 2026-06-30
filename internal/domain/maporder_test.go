package domain

import (
	"reflect"
	"testing"
)

func TestSuggestMapOrder(t *testing.T) {
	in := []string{"Muldraugh, KY", "Riverside", "West Point, KY"}
	got := SuggestMapOrder(in)
	want := []string{"Riverside", "Muldraugh, KY", "West Point, KY"}
	if !reflect.DeepEqual(got.Ordered, want) {
		t.Fatalf("Ordered=%v want %v", got.Ordered, want)
	}
	if !reflect.DeepEqual(got.Moved, []string{"Muldraugh, KY"}) {
		t.Fatalf("Moved=%v want [Muldraugh, KY]", got.Moved)
	}
	if got.Reasons["Muldraugh, KY"] == "" {
		t.Fatalf("expected a reason for the moved base map")
	}
}

func TestIsBaseMap(t *testing.T) {
	if !IsBaseMap("Muldraugh, KY") {
		t.Fatal("Muldraugh, KY should be a base map")
	}
	if IsBaseMap("Riverside") {
		t.Fatal("Riverside should not be a base map")
	}
}

func TestSuggestMapOrderEdgeCases(t *testing.T) {
	// Test empty input.
	got := SuggestMapOrder(nil)
	if len(got.Ordered) != 0 {
		t.Fatalf("empty input: Ordered=%v want empty", got.Ordered)
	}
	if len(got.Moved) != 0 {
		t.Fatalf("empty input: Moved=%v want empty/nil", got.Moved)
	}

	// Test all base maps: should keep order, have no Moved, have Reasons for both.
	in := []string{"Muldraugh, KY", "West Point, KY"}
	got = SuggestMapOrder(in)
	want := []string{"Muldraugh, KY", "West Point, KY"}
	if !reflect.DeepEqual(got.Ordered, want) {
		t.Fatalf("all base maps: Ordered=%v want %v", got.Ordered, want)
	}
	if len(got.Moved) != 0 {
		t.Fatalf("all base maps: Moved=%v want empty/nil", got.Moved)
	}
	if len(got.Reasons) != 2 {
		t.Fatalf("all base maps: Reasons count=%d want 2", len(got.Reasons))
	}
	if got.Reasons["Muldraugh, KY"] == "" || got.Reasons["West Point, KY"] == "" {
		t.Fatal("all base maps: expected reasons for both")
	}

	// Test no base maps: should keep order, have no Moved, have empty Reasons.
	in = []string{"Riverside", "CustomA"}
	got = SuggestMapOrder(in)
	want = []string{"Riverside", "CustomA"}
	if !reflect.DeepEqual(got.Ordered, want) {
		t.Fatalf("no base maps: Ordered=%v want %v", got.Ordered, want)
	}
	if len(got.Moved) != 0 {
		t.Fatalf("no base maps: Moved=%v want empty/nil", got.Moved)
	}
	if len(got.Reasons) != 0 {
		t.Fatalf("no base maps: Reasons=%v want empty", got.Reasons)
	}
}
