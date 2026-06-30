package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// menuItem is one selectable row in a menu.
type menuItem struct {
	Label string
	Desc  string
}

// menu is a simple vertical cursor list used by the menu-style screens. Feature
// screens use the richer row helpers instead.
type menu struct {
	items  []menuItem
	cursor int
}

func (m *menu) setItems(items []menuItem) {
	m.items = items
	if m.cursor >= len(items) {
		m.cursor = max(0, len(items)-1)
	}
}

func (m *menu) up() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *menu) down() {
	if m.cursor < len(m.items)-1 {
		m.cursor++
	}
}

func (m *menu) selected() int { return m.cursor }

func (m menu) View(th Theme, width int) string {
	var b strings.Builder
	for i, it := range m.items {
		sel := i == m.cursor
		if sel {
			content := "▸ " + it.Label
			if it.Desc != "" {
				content += "  " + it.Desc
			}
			b.WriteString(th.SelectedRow.Width(width).Render(stripANSI(content)) + "\n")
			continue
		}
		line := cursorPrefix(th, sel) + th.Item.Render(it.Label)
		if it.Desc != "" {
			line += "  " + th.Muted.Render(it.Desc)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// pad renders content into a left-padded body region (1-column gutter).
func pad(s string) string {
	return lipgloss.NewStyle().Padding(0, 1).Render(s)
}
