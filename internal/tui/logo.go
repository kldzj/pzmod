package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// logoArt is the 6-row ANSI-Shadow "pzmod" wordmark. Rows 1 and 6 may lose
// their trailing space when stored (editors strip trailing whitespace), so they
// are 43 cells while rows 2-5 are 44. logoHero right-pads every line to the max
// width, so the wordmark renders as a clean rectangle regardless.
const logoArt = `██████╗ ███████╗███╗   ███╗ ██████╗ ██████╗
██╔══██╗╚══███╔╝████╗ ████║██╔═══██╗██╔══██╗
██████╔╝  ███╔╝ ██╔████╔██║██║   ██║██║  ██║
██╔═══╝  ███╔╝  ██║╚██╔╝██║██║   ██║██║  ██║
██║     ███████╗██║ ╚═╝ ██║╚██████╔╝██████╔╝
╚═╝     ╚══════╝╚═╝     ╚═╝ ╚═════╝ ╚═════╝`

const (
	logoTagline  = "this is how you modded."
	sponsorURL   = "github.com/sponsors/kldzj"
	logoArtWidth = 44
)

// logoLines splits the art into lines, each right-padded to the widest line so
// the block is a clean rectangle regardless of stored trailing spaces.
func logoLines() []string {
	raw := strings.Split(logoArt, "\n")
	w := 0
	for _, l := range raw {
		if cw := runewidth.StringWidth(l); cw > w {
			w = cw
		}
	}
	out := make([]string, len(raw))
	for i, l := range raw {
		pad := w - runewidth.StringWidth(l)
		if pad < 0 {
			pad = 0
		}
		out[i] = l + strings.Repeat(" ", pad)
	}
	return out
}

// logoHero renders the framed hero: the accent wordmark, the tagline, and the
// ♥ sponsor line, inside Theme.Box (rounded accent border, no background).
func logoHero(th Theme) string {
	accent := lipgloss.NewStyle().Bold(true).Foreground(th.Accent)
	var b strings.Builder
	for i, l := range logoLines() {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(accent.Render(l))
	}
	b.WriteString("\n\n")
	b.WriteString(th.Muted.Render(logoTagline))
	b.WriteString("\n")
	b.WriteString(accent.Render("♥ ") + th.Muted.Render(sponsorURL))
	return th.Box.Render(b.String())
}

// logoCompact renders the one-line mark used on the dashboard and on small
// terminals: "◆ pzmod" in accent bold + a muted tagline.
func logoCompact(th Theme) string {
	mark := lipgloss.NewStyle().Bold(true).Foreground(th.Accent).Render("◆ pzmod")
	return mark + th.Muted.Render(" · Project Zomboid server mod manager")
}

// renderLogo returns the framed hero when the terminal is large enough,
// otherwise the compact one-line mark. The launcher uses this; the dashboard
// always calls logoCompact directly.
func renderLogo(th Theme, width, height int) string {
	if width >= logoArtWidth+8 && height >= 18 {
		return logoHero(th)
	}
	return logoCompact(th)
}
