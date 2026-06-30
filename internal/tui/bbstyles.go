package tui

import "github.com/kldzj/pzmod/internal/bbcode"

// bbStyles maps the TUI theme onto the bbcode renderer's style hooks.
func bbStyles(th Theme) bbcode.Styles {
	return bbcode.Styles{
		Bold:      th.Subtitle,
		Italic:    th.Item,
		Underline: th.Item,
		Strike:    th.Muted,
		Heading:   th.Title,
		Quote:     th.Muted,
		Code:      th.Faint,
		Muted:     th.Muted,
		Link:      func(url, text string) string { return hyperlink(url, text) },
		Bullet:    "• ",
	}
}
