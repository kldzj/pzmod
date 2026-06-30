// Package bbcode renders Steam Workshop BBCode into terminal-styled text. It is
// display-only and never errors: malformed input degrades to stripped text.
package bbcode

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles are injected by the caller so this package stays decoupled from any
// particular theme.
type Styles struct {
	Bold      lipgloss.Style
	Italic    lipgloss.Style
	Underline lipgloss.Style
	Strike    lipgloss.Style
	Heading   lipgloss.Style
	Quote     lipgloss.Style
	Code      lipgloss.Style
	Muted     lipgloss.Style
	Link      func(url, text string) string
	Bullet    string
}

var (
	reTag    = regexp.MustCompile(`(?is)\[(/?)([a-z0-9*]+)(?:=([^\]]*))?\]`)
	reSimple = map[string]func(Styles, string) string{
		"b":       func(s Styles, in string) string { return s.Bold.Render(in) },
		"i":       func(s Styles, in string) string { return s.Italic.Render(in) },
		"u":       func(s Styles, in string) string { return s.Underline.Render(in) },
		"strike":  func(s Styles, in string) string { return s.Strike.Render(in) },
		"h1":      func(s Styles, in string) string { return s.Heading.Render(in) },
		"h2":      func(s Styles, in string) string { return s.Heading.Render(in) },
		"h3":      func(s Styles, in string) string { return s.Heading.Render(in) },
		"quote":   func(s Styles, in string) string { return s.Quote.Render(in) },
		"code":    func(s Styles, in string) string { return s.Code.Render(in) },
		"spoiler": func(s Styles, in string) string { return s.Muted.Render(in) },
	}
)

// Render converts BBCode in src to styled text.
func Render(src string, st Styles) string {
	// Normalize Steam's CRLF first.
	src = strings.ReplaceAll(src, "\r\n", "\n")
	return renderRange(src, st)
}

// renderRange resolves the outermost balanced tags left to right.
func renderRange(src string, st Styles) string {
	var b strings.Builder
	for len(src) > 0 {
		loc := reTag.FindStringSubmatchIndex(src)
		if loc == nil {
			b.WriteString(src)
			break
		}
		b.WriteString(src[:loc[0]]) // text before the tag
		closing := src[loc[2]:loc[3]] == "/"
		name := strings.ToLower(src[loc[4]:loc[5]])
		arg := ""
		if loc[6] >= 0 {
			arg = src[loc[6]:loc[7]]
		}
		full := src[loc[0]:loc[1]]
		rest := src[loc[1]:]

		if closing {
			// Stray closing tag: drop it.
			src = rest
			continue
		}
		inner, after, found := splitClose(rest, name)
		if !found {
			// Unbalanced/self tag: handle the few that need it, else drop the tag.
			switch name {
			case "img":
				b.WriteString("[image: " + strings.TrimSpace(arg) + "]")
			default:
				_ = full
			}
			src = rest
			continue
		}
		b.WriteString(renderTag(name, arg, inner, st))
		src = after
	}
	return b.String()
}

// splitClose finds the matching [/name] for the current [name...], honoring
// nesting of the same tag. It scans tags with reTag and matches names exactly,
// so [i] is never confused with [img].
func splitClose(src, name string) (inner, after string, found bool) {
	depth := 1
	pos := 0
	for {
		loc := reTag.FindStringSubmatchIndex(src[pos:])
		if loc == nil {
			return "", "", false
		}
		closing := src[pos+loc[2]:pos+loc[3]] == "/"
		tname := strings.ToLower(src[pos+loc[4] : pos+loc[5]])
		tagStart := pos + loc[0]
		tagEnd := pos + loc[1]
		if tname == name {
			if closing {
				depth--
				if depth == 0 {
					return src[:tagStart], src[tagEnd:], true
				}
			} else {
				depth++
			}
		}
		pos = tagEnd
	}
}

func renderTag(name, arg, inner string, st Styles) string {
	switch name {
	case "url":
		text := renderRange(inner, st)
		target := strings.TrimSpace(arg)
		if target == "" {
			target = strings.TrimSpace(inner)
		}
		if st.Link != nil {
			return st.Link(target, text)
		}
		return text
	case "img":
		return "[image: " + strings.TrimSpace(inner) + "]"
	case "previewyoutube":
		return "[video: " + strings.TrimSpace(arg) + "]"
	case "list", "olist":
		return renderList(inner, st)
	}
	if fn, ok := reSimple[name]; ok {
		return fn(st, renderRange(inner, st))
	}
	// Unknown tag: keep inner text.
	return renderRange(inner, st)
}

// renderList turns [*] items into bulleted lines.
func renderList(inner string, st Styles) string {
	parts := strings.Split(inner, "[*]")
	var b strings.Builder
	b.WriteString("\n")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		b.WriteString(st.Bullet + renderRange(p, st) + "\n")
	}
	return b.String()
}
