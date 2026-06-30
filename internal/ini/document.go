// Package ini implements a byte-exact, round-trip-preserving INI document model.
//
// Unlike a typical INI library, it does not normalize input: comments, blank
// lines, key order, inline content, and the original line endings are all
// preserved. An unmodified document renders byte-for-byte identical to its
// source; only entries changed via Set are re-rendered. The package has no
// knowledge of Project Zomboid - see internal/serverconfig for that.
package ini

import "io"

// Document is an ordered sequence of lines with a key index for fast lookup.
type Document struct {
	lines    []Line
	index    map[string]int // key -> first line index
	eol      LineEnding     // dominant ending, used when appending new lines
	original []byte         // bytes the document was parsed from (for dirty checks)
}

// Parse builds a Document from raw bytes, preserving everything verbatim.
func Parse(data []byte) *Document {
	d := &Document{
		index:    make(map[string]int),
		eol:      detectEOL(string(data), OSDefault()),
		original: append([]byte(nil), data...),
	}

	for _, rl := range splitLines(string(data)) {
		line := parseLine(rl.text, rl.term)
		if line.Kind == KindEntry {
			if _, seen := d.index[line.key]; !seen {
				d.index[line.key] = len(d.lines)
			}
		}
		d.lines = append(d.lines, line)
	}
	return d
}

// New returns an empty document using the OS-default line ending.
func New() *Document {
	return &Document{index: make(map[string]int), eol: OSDefault(), original: []byte{}}
}

// EOL returns the document's dominant line ending.
func (d *Document) EOL() LineEnding { return d.eol }

// Lines returns the document's lines (read-only use).
func (d *Document) Lines() []Line { return d.lines }

// Has reports whether key is present.
func (d *Document) Has(key string) bool {
	_, ok := d.index[key]
	return ok
}

// Get returns the logical value for key and whether it was found.
func (d *Document) Get(key string) (string, bool) {
	i, ok := d.index[key]
	if !ok {
		return "", false
	}
	return d.lines[i].Value(), true
}

// GetOr returns the value for key, or def when absent.
func (d *Document) GetOr(key, def string) string {
	if v, ok := d.Get(key); ok {
		return v
	}
	return def
}

// Set updates an existing key in place (preserving its surrounding lines) or
// appends a new "key=value" entry using the document's line ending.
func (d *Document) Set(key, value string) {
	if i, ok := d.index[key]; ok {
		d.lines[i].dirty = true
		d.lines[i].value = value
		return
	}

	// Ensure the previous final line is newline-terminated before appending,
	// so a file with no trailing newline doesn't get two entries on one line.
	if n := len(d.lines); n > 0 && d.lines[n-1].term == "" {
		d.lines[n-1].term = d.eol.String()
	}

	d.index[key] = len(d.lines)
	d.lines = append(d.lines, Line{
		Kind:  KindEntry,
		key:   key,
		dirty: true,
		value: value,
		term:  d.eol.String(),
	})
}

// Bytes renders the document back to bytes.
func (d *Document) Bytes() []byte {
	var b []byte
	for i := range d.lines {
		b = append(b, d.lines[i].render()...)
	}
	return b
}

// String renders the document back to a string.
func (d *Document) String() string { return string(d.Bytes()) }

// WriteTo implements io.WriterTo.
func (d *Document) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(d.Bytes())
	return int64(n), err
}

// HasUnsavedChanges reports whether the in-memory document differs from the
// bytes it was parsed from.
func (d *Document) HasUnsavedChanges() bool {
	return string(d.Bytes()) != string(d.original)
}

// MarkSaved records the current rendering as the clean baseline. Call this
// after persisting the document so HasUnsavedChanges resets.
func (d *Document) MarkSaved() {
	d.original = d.Bytes()
	for i := range d.lines {
		if d.lines[i].dirty {
			// Bake the rendered value (incl. any inline comment) into raw so
			// future renders stay stable.
			raw := d.lines[i].key + "=" + d.lines[i].value
			if d.lines[i].inlineComment != "" {
				raw += " " + d.lines[i].inlineComment
			}
			d.lines[i].raw = raw
			d.lines[i].dirty = false
		}
	}
}

type rawLine struct {
	text string
	term string
}

// splitLines splits content into lines, capturing each line's exact terminator
// ("\n", "\r\n", "\r", or "" for a final line without a newline). This makes
// rendering byte-exact even for mixed endings or a missing trailing newline.
func splitLines(s string) []rawLine {
	var out []rawLine
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && s[j] != '\n' && s[j] != '\r' {
			j++
		}
		text := s[i:j]
		term := ""
		if j < len(s) {
			if s[j] == '\r' {
				if j+1 < len(s) && s[j+1] == '\n' {
					term = "\r\n"
					j += 2
				} else {
					term = "\r"
					j++
				}
			} else {
				term = "\n"
				j++
			}
		}
		out = append(out, rawLine{text: text, term: term})
		i = j
	}
	return out
}
