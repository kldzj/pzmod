package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/steam/steamtest"
)

func TestInstalledFilterByTitle(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100100",
			Title: "Hydrocraft", Description: "Mod ID: Hydrocraft\n"},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200200",
			Title: "Brita Weapons", Description: "Mod ID: BWPack\n"},
	)
	tm, _ := openProfileModelWith(t, fake, "WorkshopItems=100100;200200\nMods=Hydrocraft;BWPack\nMap=\n")
	tm.Send(PushMsg{Screen: NewInstalled()})
	waitForText(t, tm, "Brita Weapons")

	tm.Send(keyRune('/'))
	for _, r := range "brita" {
		tm.Send(keyRune(r))
	}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("filter: brita")) && bytes.Contains(b, []byte("Brita Weapons"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestInstalledFilterMatchesModID(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100100",
			Title: "Hydrocraft", Description: "Mod ID: Hydrocraft\n"},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200200",
			Title: "Brita Weapons", Description: "Mod ID: BWPack\n"},
	)
	tm, _ := openProfileModelWith(t, fake, "WorkshopItems=100100;200200\nMods=Hydrocraft;BWPack\nMap=\n")
	tm.Send(PushMsg{Screen: NewInstalled()})
	waitForText(t, tm, "Brita Weapons")

	tm.Send(keyRune('/'))
	for _, r := range "bwpack" {
		tm.Send(keyRune(r))
	}
	// "bwpack" matches the declared mod ID, not the title "Brita Weapons".
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("filter: bwpack")) && bytes.Contains(b, []byte("Brita Weapons"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestInstalledFilterEscClearsThenPops(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100100",
			Title: "Hydrocraft", Description: "Mod ID: Hydrocraft\n"},
	)
	tm, _ := openProfileModelWith(t, fake, "WorkshopItems=100100\nMods=Hydrocraft\nMap=\n")
	tm.Send(PushMsg{Screen: NewInstalled()})
	waitForText(t, tm, "Hydrocraft")

	tm.Send(keyRune('/'))
	tm.Send(keyRune('z')) // no matches
	waitForText(t, tm, "no matches for")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc}) // clears the filter -> list returns
	waitForText(t, tm, "Hydrocraft")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})                 // pops back to the dashboard
	waitForText(t, tm, "check dependencies and problems") // a dashboard-only menu desc

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
