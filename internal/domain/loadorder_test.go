package domain

import (
	"reflect"
	"testing"
)

func TestTopoOrderDependencyFirst(t *testing.T) {
	// b depends on a, but a is listed second; a must move first.
	plan := TopoOrder([]string{"b", "a"}, map[string][]string{"b": {"a"}}, nil)
	if !reflect.DeepEqual(plan.Ordered, []string{"a", "b"}) {
		t.Errorf("Ordered = %v; want [a b]", plan.Ordered)
	}
	if len(plan.Cycles) != 0 {
		t.Errorf("unexpected cycles: %v", plan.Cycles)
	}
}

func TestTopoOrderFrameworkFirst(t *testing.T) {
	// No edges, but b is a framework mod, so it loads first.
	plan := TopoOrder([]string{"a", "b", "c"}, nil, map[string]bool{"b": true})
	if plan.Ordered[0] != "b" {
		t.Errorf("Ordered = %v; want b first", plan.Ordered)
	}
}

func TestTopoOrderStableWhenSatisfied(t *testing.T) {
	// Already valid order with no constraints must be left untouched.
	in := []string{"a", "b", "c"}
	plan := TopoOrder(in, nil, nil)
	if !reflect.DeepEqual(plan.Ordered, in) {
		t.Errorf("Ordered = %v; want unchanged %v", plan.Ordered, in)
	}
	if len(plan.Moved) != 0 {
		t.Errorf("Moved = %v; want none", plan.Moved)
	}
}

func TestTopoOrderCycleReported(t *testing.T) {
	plan := TopoOrder([]string{"x", "y"}, map[string][]string{"x": {"y"}, "y": {"x"}}, nil)
	// Cycle members are still emitted (original order), and the cycle is reported.
	if len(plan.Ordered) != 2 {
		t.Errorf("Ordered = %v; want both nodes present", plan.Ordered)
	}
	if len(plan.Cycles) == 0 {
		t.Error("expected a cycle to be reported")
	}
}

func TestDetectCyclesNone(t *testing.T) {
	if c := DetectCycles(map[string][]string{"a": {"b"}, "b": {"c"}}); len(c) != 0 {
		t.Errorf("DetectCycles = %v; want none", c)
	}
}
