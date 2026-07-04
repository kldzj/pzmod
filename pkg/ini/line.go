package ini

import "strings"

// Kind classifies a parsed line.
type Kind int

const (
	// KindBlank is an empty or whitespace-only line.
	KindBlank Kind = iota
	// KindComment is a line whose first non-space rune is '#'.
	KindComment
	// KindEntry is a "key=value" line.
	KindEntry
	// KindOther is a non-blank, non-comment line with no '=' (preserved verbatim).
	KindOther
)

// Line is one physical line of a Document. It retains the exact original bytes
// (raw text + terminator) so that an unmodified Document renders byte-for-byte
// identical to its source. Only a line mutated via Document.Set is re-rendered.
type Line struct {
	Kind Kind

	raw  string // line text without the trailing EOL
	term string // "\n", "\r\n", "\r", or "" for a final line with no newline

	key           string // parsed key, for KindEntry
	inlineComment string // trailing "# ..." comment on an entry line, if any

	dirty bool   // set when the value was replaced via Document.Set
	value string // replacement value, valid only when dirty
}

// Key returns the entry key, or "" for non-entry lines.
func (l *Line) Key() string { return l.key }

// Raw returns the original line text (without its terminator).
func (l *Line) Raw() string { return l.raw }

// Value returns the logical value: the part after the first '=', with any inline
// comment removed and surrounding whitespace trimmed. For a mutated line it
// returns the replacement.
func (l *Line) Value() string {
	if l.Kind != KindEntry {
		return ""
	}
	if l.dirty {
		return l.value
	}
	v, _ := splitValueComment(valuePart(l.raw))
	return v
}

// render reproduces the on-disk bytes for this line. A mutated entry keeps its
// inline comment so editing a value never silently drops the note beside it.
func (l *Line) render() string {
	if l.Kind == KindEntry && l.dirty {
		out := l.key + "=" + l.value
		if l.inlineComment != "" {
			out += " " + l.inlineComment
		}
		return out + l.term
	}
	return l.raw + l.term
}

// parseLine classifies a raw line (without terminator) and extracts its key and
// any inline comment.
func parseLine(raw, term string) Line {
	l := Line{raw: raw, term: term}

	trimmed := strings.TrimSpace(raw)
	switch {
	case trimmed == "":
		l.Kind = KindBlank
	case strings.HasPrefix(trimmed, "#"):
		l.Kind = KindComment
	case strings.ContainsRune(raw, '='):
		l.Kind = KindEntry
		l.key = strings.TrimSpace(raw[:strings.IndexByte(raw, '=')])
		_, l.inlineComment = splitValueComment(valuePart(raw))
	default:
		l.Kind = KindOther
	}
	return l
}

// valuePart returns everything after the first '=' (or "" if none).
func valuePart(raw string) string {
	i := strings.IndexByte(raw, '=')
	if i < 0 {
		return ""
	}
	return raw[i+1:]
}

// splitValueComment separates a value from a trailing inline comment. An inline
// comment begins at the first '#' that is preceded by whitespace, e.g.
// "Name   # note" -> ("Name", "# note"). A leading '#' with no preceding
// whitespace (e.g. a "#hexvalue") is treated as part of the value, not a comment.
func splitValueComment(s string) (value, comment string) {
	for j := 0; j < len(s); j++ {
		if s[j] == '#' && j > 0 && (s[j-1] == ' ' || s[j-1] == '\t') {
			return strings.TrimSpace(s[:j]), strings.TrimSpace(s[j:])
		}
	}
	return strings.TrimSpace(s), ""
}
