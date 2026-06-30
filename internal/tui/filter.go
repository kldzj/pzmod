package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// filterState is an inline type-to-filter for list screens, mirroring the
// established Backups behavior: '/' to start typing, printable runes to type,
// backspace to delete, enter to apply (keep the query, stop typing), esc to
// clear and stop. Navigation keys are intentionally NOT consumed here, so a
// caller can still move the cursor while typing (it calls handleKey first and
// falls through to its own nav switch when handleKey returns false).
type filterState struct {
	active bool   // currently capturing keystrokes
	query  string // applied filter text; may persist after typing stops
}

func (f *filterState) start() { f.active = true }

func (f *filterState) clear() { f.active, f.query = false, "" }

func (f *filterState) has() bool { return f.query != "" }

// handleKey processes a key while typing. It returns true when it consumed the
// key: esc clears; enter stops typing but keeps the query; backspace deletes the
// last rune; printable runes are appended. Arrows/pgup/etc. are not consumed.
func (f *filterState) handleKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "esc":
		f.clear()
		return true
	case "enter":
		f.active = false
		return true
	case "backspace":
		if r := []rune(f.query); len(r) > 0 {
			f.query = string(r[:len(r)-1])
		}
		return true
	}
	if len(msg.Runes) > 0 {
		f.query += string(msg.Runes)
		return true
	}
	return false
}

// view renders the "filter: <query>▌" line (the ▌ cursor only while typing).
// Returns "" when there is nothing to show.
func (f *filterState) view(th Theme) string {
	if !f.active && f.query == "" {
		return ""
	}
	line := th.Muted.Render("filter: ") + f.query
	if f.active {
		line += th.Title.Render("▌")
	}
	return line
}

// chrome reports how many body lines view() occupies (line + blank separator),
// for height math. 0 when nothing is shown.
func (f *filterState) chrome() int {
	if !f.active && f.query == "" {
		return 0
	}
	return 2
}

// filterMatch reports whether query (case-insensitive) is a substring of any of
// the provided fields. An empty query matches everything.
func filterMatch(query string, fields ...string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}
