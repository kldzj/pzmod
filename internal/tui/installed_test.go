package tui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/steam/steamtest"
)

func TestInstalledScrollsToEnd(t *testing.T) {
	const n = 30
	items := make([]steam.WorkshopItem, n)
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%d", i+1)
		ids[i] = id
		items[i] = steam.WorkshopItem{
			Result:          1,
			FileType:        steam.FileTypeMod,
			PublishedFileID: id,
			Title:           fmt.Sprintf("Mod-%02d-Title", i+1),
			Description:     fmt.Sprintf("Mod ID: Mod%02d\n", i+1),
		}
	}
	fake := steamtest.New(items...)
	iniContent := "WorkshopItems=" + strings.Join(ids, ";") + "\nMods=\n"

	tm, _ := openProfileModelWith(t, fake, iniContent)
	tm.Send(PushMsg{Screen: NewInstalled()})

	// Wait for items to load AND verify the hint is not clipped in the initial full-frame
	// render. Bubbletea's partial renderer does not re-emit unchanged lines in subsequent
	// frames, so "esc: back" (hint, line 26) would be absent from the End-key diff if its
	// position is unchanged. We assert both in this first full frame where both must appear.
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Mod-01-Title")) && bytes.Contains(b, []byte("esc: back"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	// Jump to the last item - without windowing the bottom items are clipped.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnd})
	waitForText(t, tm, "Mod-30-Title")
}

// TestInstalledAOpensAddByID verifies that pressing 'a' on the Installed screen
// opens the Add-by-ID input screen, even when the list is empty.
func TestInstalledAOpensAddByID(t *testing.T) {
	fake := steamtest.New()
	tm, _ := openProfileModelWith(t, fake, "WorkshopItems=\nMods=\nMap=\n")
	tm.Send(PushMsg{Screen: NewInstalled()})
	waitForText(t, tm, "nothing installed") // empty state renders

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	waitForText(t, tm, "Workshop ID") // AddByID placeholder text
}

func TestInstalledListsAndRemoves(t *testing.T) {
	fake := steamtest.New()
	fake.Items["100"] = steam.WorkshopItem{PublishedFileID: "100", Title: "Basements", Description: "Mod ID: Basements\nMap Folder: BaseTiles"}

	tm, m := openProfileModelWith(t, fake, "WorkshopItems=100\nMods=Basements\nMap=BaseTiles\n")
	tm.Send(PushMsg{Screen: NewInstalled()})
	waitForText(t, tm, "Basements")

	// Remove -> confirm -> y
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	waitForText(t, tm, "Remove")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	waitForText(t, tm, "removed")

	sm := m.s.Cfg.ServerMods()
	if sm.HasItem("100") || sm.HasMod("Basements") || sm.HasMap("BaseTiles") {
		t.Fatalf("expected item+owned mod+map removed: %+v", sm)
	}
}
