package bbcode

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func testStyles() Styles {
	id := lipgloss.NewStyle()
	return Styles{
		Bold: id, Italic: id, Underline: id, Strike: id,
		Heading: id, Quote: id, Code: id, Muted: id,
		Link:   func(url, text string) string { return text + " <" + url + ">" },
		Bullet: "• ",
	}
}

func TestRenderStripsUnknownAndKeepsText(t *testing.T) {
	out := Render("[unknown]hello[/unknown] world", testStyles())
	if !strings.Contains(out, "hello world") {
		t.Fatalf("expected text preserved, got %q", out)
	}
}

func TestRenderURL(t *testing.T) {
	out := Render("see [url=https://x.test]here[/url]", testStyles())
	if !strings.Contains(out, "here <https://x.test>") {
		t.Fatalf("url not rendered: %q", out)
	}
}

func TestRenderListBullets(t *testing.T) {
	out := Render("[list][*]one[*]two[/list]", testStyles())
	if !strings.Contains(out, "• one") || !strings.Contains(out, "• two") {
		t.Fatalf("list not rendered: %q", out)
	}
}

func TestRenderImagePlaceholder(t *testing.T) {
	out := Render("[img]https://x.test/a.png[/img]", testStyles())
	if !strings.Contains(out, "[image: https://x.test/a.png]") {
		t.Fatalf("img not rendered: %q", out)
	}
}

// Robustness tests guarding the nesting fix.

// TestRenderNestedDifferentPrefixTag ensures [img] inside [i] is not mistaken
// as a nested [i] due to prefix matching (the bug the corrected splitClose fixes).
func TestRenderNestedDifferentPrefixTag(t *testing.T) {
	out := Render("[i]a[img]https://x.test/p.png[/img]b[/i]", testStyles())
	if !strings.Contains(out, "a") {
		t.Fatalf("expected 'a' in output, got %q", out)
	}
	if !strings.Contains(out, "b") {
		t.Fatalf("expected 'b' in output, got %q", out)
	}
	if !strings.Contains(out, "[image: https://x.test/p.png]") {
		t.Fatalf("expected image placeholder in output, got %q", out)
	}
}

// TestRenderNestedSameTag ensures nested same-tag pairs are balanced correctly.
func TestRenderNestedSameTag(t *testing.T) {
	out := Render("[b]x[b]y[/b]z[/b]", testStyles())
	if !strings.Contains(out, "x") {
		t.Fatalf("expected 'x' in output, got %q", out)
	}
	if !strings.Contains(out, "y") {
		t.Fatalf("expected 'y' in output, got %q", out)
	}
	if !strings.Contains(out, "z") {
		t.Fatalf("expected 'z' in output, got %q", out)
	}
	if strings.Contains(out, "[b]") || strings.Contains(out, "[/b]") {
		t.Fatalf("literal tags leaked into output: %q", out)
	}
}

// TestRenderUnbalanced ensures an unbalanced open tag degrades gracefully.
func TestRenderUnbalanced(t *testing.T) {
	// Must not panic and must preserve the text content.
	out := Render("[b]oops", testStyles())
	if !strings.Contains(out, "oops") {
		t.Fatalf("expected 'oops' in output, got %q", out)
	}
}
