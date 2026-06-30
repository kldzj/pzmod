package tui

import (
	"testing"

	"github.com/kldzj/pzmod/internal/steam/steamtest"
)

func TestDashboardAutoValidates(t *testing.T) {
	tm, _ := openProfileModelWith(t, steamtest.New(), "Mods=Ghost\nWorkshopItems=\nMap=\n")
	// Ghost mod with no backing item produces a finding (or a clean result);
	// either way the status line updates from "not validated yet" to one containing "last check".
	waitForText(t, tm, "last check")
}
