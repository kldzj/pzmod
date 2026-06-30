package tui

import (
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/mattn/go-runewidth"
)

// ansiPattern matches SGR escape sequences (colors/attributes).
var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

// stripANSI removes SGR escapes so a string can be re-styled (e.g. given a
// selection background) without its own embedded resets punching holes in the
// new background. lipgloss applies the outer style once at the start; any inner
// reset would clear it for the remainder of the line.
func stripANSI(s string) string { return ansiPattern.ReplaceAllString(s, "") }

// renderRow draws a single full-width list row: an optional (already-styled)
// prefix, a left label (truncated to fit), and right-aligned meta. When selected
// the whole row gets the selection background.
func renderRow(th Theme, width int, prefix, left, right string, selected bool) string {
	if width < 12 {
		width = 12
	}
	prefixW := lipgloss.Width(prefix)
	rightW := runewidth.StringWidth(right)
	const gap = 2
	leftMax := width - prefixW - rightW - gap
	if leftMax < 1 {
		leftMax = 1
	}
	left = runewidth.Truncate(left, leftMax, "…")
	leftW := runewidth.StringWidth(left)
	pad := width - prefixW - leftW - rightW
	if pad < 1 {
		pad = 1
	}

	if selected {
		// The selection paints a full-width background. Flatten to plain text so
		// the styled segments' resets don't punch holes in it; SelectedRow owns
		// all coloring for the row.
		content := stripANSI(prefix) + stripANSI(left) + strings.Repeat(" ", pad) + stripANSI(right)
		return th.SelectedRow.Width(width).Render(content)
	}

	var b strings.Builder
	b.WriteString(prefix)
	b.WriteString(th.Item.Render(left))
	b.WriteString(strings.Repeat(" ", pad))
	b.WriteString(th.Muted.Render(right))
	return b.String()
}

// cursorPrefix returns the leading marker for a row.
func cursorPrefix(th Theme, selected bool) string {
	if selected {
		return th.Title.Render("▸ ")
	}
	return "  "
}

// chips renders a row of labelled chips.
func chips(th Theme, labels ...string) string {
	parts := make([]string, 0, len(labels))
	for _, l := range labels {
		if l = strings.TrimSpace(l); l != "" {
			parts = append(parts, th.Chip.Render(l))
		}
	}
	return strings.Join(parts, " ")
}

// hyperlink wraps text in an OSC-8 terminal hyperlink. Terminals that support it
// render text as clickable; others just show the text.
func hyperlink(url, text string) string {
	if url == "" {
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// relTime renders a Unix timestamp as a relative string, e.g. "3 days ago".
func relTime(unix int64) string {
	if unix <= 0 {
		return ""
	}
	return humanize.Time(time.Unix(unix, 0))
}

// relTimeRFC renders an RFC3339 timestamp as a relative string.
func relTimeRFC(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return humanize.Time(t)
}

// metaLine joins non-empty fields with a middot separator.
func metaLine(fields ...string) string {
	var out []string
	for _, f := range fields {
		if strings.TrimSpace(f) != "" {
			out = append(out, f)
		}
	}
	return strings.Join(out, " · ")
}

// listWindow returns the [start,end) range of indices to render so that a
// viewport of `height` rows keeps `cursor` visible, scrolling as the cursor
// nears the edges.
func listWindow(cursor, total, height int) (start, end int) {
	if height < 1 {
		height = 1
	}
	if cursor >= total {
		cursor = total - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	if total <= height {
		return 0, total
	}
	if cursor >= height {
		start = cursor - height + 1
	}
	if start > total-height {
		start = total - height
	}
	if start < 0 {
		start = 0
	}
	end = start + height
	if end > total {
		end = total
	}
	return start, end
}
