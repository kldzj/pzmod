package tui

import "testing"

func TestListWindow(t *testing.T) {
	t.Run("total<=height returns (0,total)", func(t *testing.T) {
		for _, tc := range []struct{ total, height int }{{0, 10}, {5, 10}, {10, 10}} {
			s, e := listWindow(0, tc.total, tc.height)
			if s != 0 || e != tc.total {
				t.Errorf("total=%d height=%d: got (%d,%d); want (0,%d)", tc.total, tc.height, s, e, tc.total)
			}
		}
	})

	t.Run("cursor near top stays at start=0", func(t *testing.T) {
		s, e := listWindow(2, 20, 10)
		if s != 0 || e != 10 {
			t.Errorf("got (%d,%d); want (0,10)", s, e)
		}
	})

	t.Run("cursor at end yields end==total and start==total-height", func(t *testing.T) {
		s, e := listWindow(19, 20, 10)
		if s != 10 || e != 20 {
			t.Errorf("got (%d,%d); want (10,20)", s, e)
		}
	})

	t.Run("mid-list keeps cursor in [start,end)", func(t *testing.T) {
		cursor := 15
		s, e := listWindow(cursor, 30, 10)
		if cursor < s || cursor >= e {
			t.Errorf("cursor=%d not in [%d,%d)", cursor, s, e)
		}
	})

	t.Run("height=1 always shows only cursor", func(t *testing.T) {
		for _, cursor := range []int{0, 5, 19} {
			s, e := listWindow(cursor, 20, 1)
			if s != cursor || e != cursor+1 {
				t.Errorf("cursor=%d: got (%d,%d); want (%d,%d)", cursor, s, e, cursor, cursor+1)
			}
		}
	})

	t.Run("cursor>=total is clamped and window contains it", func(t *testing.T) {
		// cursor=50 is out of range for total=10, height=5 → clamped to 9 → window [5,10)
		s, e := listWindow(50, 10, 5)
		if s != 5 || e != 10 {
			t.Errorf("got (%d,%d); want (5,10)", s, e)
		}
		if 9 < s || 9 >= e {
			t.Errorf("clamped cursor 9 not in [%d,%d)", s, e)
		}
	})
}
