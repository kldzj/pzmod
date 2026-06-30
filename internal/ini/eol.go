package ini

// Line-ending detection, merged into the ini package (formerly the standalone
// eol/ package). Original detection logic adapted from nochso/gomd.

import "runtime"

// LineEnding represents a newline combination.
type LineEnding int

const (
	LF LineEnding = iota + 1
	CR
	CRLF
)

var eolDescriptions = map[LineEnding]string{
	LF:   "LF (Unix)",
	CR:   "CR (Mac)",
	CRLF: "CRLF (Windows)",
}

var eolStrings = map[LineEnding]string{
	LF:   "\n",
	CR:   "\r",
	CRLF: "\r\n",
}

// Description returns a human-readable label, e.g. for a settings screen.
func (le LineEnding) Description() string {
	if d, ok := eolDescriptions[le]; ok {
		return d
	}
	return "Unknown"
}

// String returns the actual, unescaped line ending.
func (le LineEnding) String() string {
	if s, ok := eolStrings[le]; ok {
		return s
	}
	return "\n"
}

// OSDefault returns the preferred line ending for the current OS.
func OSDefault() LineEnding {
	if runtime.GOOS == "windows" {
		return CRLF
	}
	return LF
}

// detectEOL returns the most frequent line ending in content. When content
// contains no line endings it returns def.
func detectEOL(content string, def LineEnding) LineEnding {
	var lf, cr, crlf int
	for i := 0; i < len(content); i++ {
		switch content[i] {
		case '\r':
			if i+1 < len(content) && content[i+1] == '\n' {
				crlf++
				i++
			} else {
				cr++
			}
		case '\n':
			lf++
		}
	}

	if lf+cr+crlf == 0 {
		return def
	}
	if crlf >= lf && crlf >= cr {
		return CRLF
	}
	if cr > lf {
		return CR
	}
	return LF
}
