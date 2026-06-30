package tui

import (
	"context"
	"strings"
	"testing"
)

// On dev/test builds no version is embedded, so the background update check must
// be a no-op - this guarantees `go test` never reaches the network.
func TestCheckUpdateCmdDevBuildIsNoop(t *testing.T) {
	if msg := checkUpdateCmd()(); msg != nil {
		t.Fatalf("checkUpdateCmd() = %v on dev build; want nil (no network)", msg)
	}
}

func TestTopBarUpdateHint(t *testing.T) {
	m := New(nil, nil, context.Background(), NewDashboard())
	m.s.Width = 100

	if got := m.renderTopBar(); strings.Contains(got, "↑") {
		t.Fatalf("top bar shows an update hint without UpdateLatest set: %q", got)
	}

	m.s.UpdateLatest = "v9.9.9"
	got := m.renderTopBar()
	if !strings.Contains(got, "v9.9.9") || !strings.Contains(got, "↑") {
		t.Errorf("top bar missing update hint; got %q", got)
	}
}

// The updateAvailableMsg handler records the version and raises a toast.
func TestUpdateAvailableMsgSetsHintAndToast(t *testing.T) {
	m := New(nil, nil, context.Background(), NewDashboard())
	m.Update(updateAvailableMsg{latest: "v9.9.9"})
	if m.s.UpdateLatest != "v9.9.9" {
		t.Errorf("UpdateLatest = %q; want v9.9.9", m.s.UpdateLatest)
	}
	if !strings.Contains(m.toast, "v9.9.9") || m.toastErr {
		t.Errorf("expected a non-error toast mentioning the version; got %q (err=%v)", m.toast, m.toastErr)
	}
}
