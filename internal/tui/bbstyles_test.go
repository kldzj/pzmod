package tui

import (
	"strings"
	"testing"

	"github.com/kldzj/pzmod/internal/bbcode"
)

func TestBBStylesRendersLink(t *testing.T) {
	st := bbStyles(DefaultTheme())
	out := bbcode.Render("[url=https://x.test]go[/url]", st)
	if !strings.Contains(out, "go") || !strings.Contains(out, "https://x.test") {
		t.Fatalf("link not rendered: %q", out)
	}
}
