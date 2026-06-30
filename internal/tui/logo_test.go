package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/internal/steam/steamtest"
)

func TestLogoArtDimensions(t *testing.T) {
	lines := strings.Split(logoArt, "\n")
	if len(lines) != 6 {
		t.Fatalf("logoArt: got %d lines, want 6", len(lines))
	}
	maxW := 0
	for _, l := range lines {
		w := lipgloss.Width(l)
		if w > maxW {
			maxW = w
		}
		if w > logoArtWidth {
			t.Errorf("line %q width %d exceeds logoArtWidth %d", l, w, logoArtWidth)
		}
	}
	if maxW != logoArtWidth {
		t.Errorf("max art width = %d; want logoArtWidth %d", maxW, logoArtWidth)
	}
}

func TestLogoHeroIsRectangular(t *testing.T) {
	out := logoHero(DefaultTheme())
	lines := strings.Split(out, "\n")
	if len(lines) < 8 {
		t.Fatalf("hero too short: %d lines", len(lines))
	}
	w := lipgloss.Width(lines[0])
	for i, l := range lines {
		if lipgloss.Width(l) != w {
			t.Errorf("hero line %d width %d != %d (not rectangular)", i, lipgloss.Width(l), w)
		}
	}
}

func TestLogoHeroContainsTaglineAndSponsor(t *testing.T) {
	out := logoHero(DefaultTheme())
	if !strings.Contains(out, logoTagline) {
		t.Errorf("hero missing tagline %q", logoTagline)
	}
	if !strings.Contains(out, sponsorURL) {
		t.Errorf("hero missing sponsor URL %q", sponsorURL)
	}
}

func TestRenderLogoSelectsVariant(t *testing.T) {
	th := DefaultTheme()
	big := renderLogo(th, 100, 30)
	if !strings.Contains(big, "╮") {
		t.Errorf("large size should render the framed hero (box border):\n%s", big)
	}
	small := renderLogo(th, 40, 12)
	if !strings.Contains(small, "◆ pzmod") {
		t.Errorf("small size should render the compact mark:\n%s", small)
	}
	if strings.Contains(small, "╮") {
		t.Errorf("small size should NOT render the box border")
	}
}

func TestLauncherShowsHeroLogo(t *testing.T) {
	tm, _, _ := testModel(t, steamtest.New())
	// A distinctive slice of the ANSI-Shadow wordmark (the "z" glyph, row 2).
	waitForText(t, tm, "╚══███╔╝")
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestDashboardShowsCompactLogo(t *testing.T) {
	tm, _, _ := testModel(t, steamtest.New())
	waitForText(t, tm, "Demo Server")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForText(t, tm, "◆ pzmod") // compact mark on the dashboard
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
