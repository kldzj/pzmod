package domain

import "testing"

func TestListDelta(t *testing.T) {
	cases := []struct {
		name     string
		old, new []string
		want     Delta
	}{
		{"identical", []string{"a", "b"}, []string{"a", "b"}, Delta{}},
		{"added", []string{"a"}, []string{"a", "b"}, Delta{Added: 1}},
		{"removed", []string{"a", "b"}, []string{"a"}, Delta{Removed: 1}},
		{"reordered", []string{"a", "b"}, []string{"b", "a"}, Delta{Reordered: true}},
		{"add+remove+reorder", []string{"a", "b", "c"}, []string{"c", "a", "d"}, Delta{Added: 1, Removed: 1, Reordered: true}},
		{"empty both", nil, nil, Delta{}},
		{"added from nil", nil, []string{"a"}, Delta{Added: 1}},
		{"removed to nil", []string{"a"}, nil, Delta{Removed: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ListDelta(tc.old, tc.new)
			if got != tc.want {
				t.Fatalf("ListDelta(%v,%v)=%+v want %+v", tc.old, tc.new, got, tc.want)
			}
		})
	}
}
