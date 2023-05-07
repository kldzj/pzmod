package eol

// Many thanks to nochso for the original code.
// https://github.com/nochso/gomd/blob/1785d26cc410/eol/eol.go

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// LineEnding represents a newline combination.
type LineEnding int

// Constants used by LineEnding.
const (
	LF = iota + 1
	CR
	CRLF
)

// Descriptions can be used for human eyes, e.g. HTML options.
// Index keys match the constant values.
var Descriptions = map[int]string{
	LF:   "LF (Unix)",
	CR:   "CR (Mac)",
	CRLF: "CRLF (Win)",
}

var combos = []string{
	"\n",
	"\r",
	"\r\n",
}

// Description for consumption by humans.
func (le LineEnding) Description() string {
	if !le.isValid() {
		return "Unknown"
	}
	return Descriptions[int(le)]
}

// String returns the actual, unescaped line ending.
func (le LineEnding) String() string {
	if !le.isValid() {
		return ""
	}
	return combos[le-1]
}

func (le LineEnding) isValid() bool {
	return le >= 1 && int(le) <= len(combos)
}

// Apply a LineEnding to a string.
// This will fail if the current LineEnding can't be detected.
func (to LineEnding) Apply(text string) (string, error) {
	from, err := Detect(text)
	if err != nil {
		return "", err
	}
	return from.ConvertTo(text, to)
}

// ConvertTo a LineEnding from a known LineEnding.
func (from LineEnding) ConvertTo(text string, to LineEnding) (string, error) {
	if !to.isValid() {
		return "", errors.New(fmt.Sprintf("Invalid target line ending (integer value: %d)", to))
	}
	if !from.isValid() {
		return "", errors.New(fmt.Sprintf("Invalid source line ending (integer value: %d)", from))
	}
	lines := strings.Split(text, from.String())
	for i, line := range lines {
		lines[i] = strings.Trim(line, "\r\n")
	}
	return strings.Join(lines, to.String()), nil
}

// Detect the line ending type of a string.
//
// The most occuring combination is picked.
// An error is returned when newlines are missing from the content.
func Detect(content string) (le LineEnding, err error) {
	prev := ' '
	counts := [4]int{}
	for _, c := range content {
		if c == '\r' && prev != '\n' {
			counts[CR]++
		} else if c == '\n' {
			if prev == '\r' {
				counts[CRLF]++
			} else {
				counts[LF]++
			}
		}
		prev = c
	}
	if counts[CRLF]+counts[CR]+counts[LF] == 0 {
		err = errors.New("Unable to detect line endings (string does not contain any)")
		return
	}
	if counts[CRLF] >= counts[CR] && counts[CRLF] >= counts[LF] {
		le = CRLF
	} else if counts[CR] > counts[LF] {
		le = CR
	} else {
		le = LF
	}
	return
}

// DetectDefault returns a default line ending where Detect would return an error.
func DetectDefault(content string, defaultEnding LineEnding) (le LineEnding) {
	le, err := Detect(content)
	if err != nil {
		le = defaultEnding
	}
	return
}

// OSDefault returns the preferred line ending for the current OS.
// Most likely useless.
func OSDefault() LineEnding {
	if runtime.GOOS == "windows" {
		return CRLF
	}
	return LF
}
