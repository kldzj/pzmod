package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/kldzj/pzmod/internal/serverconfig"
	"github.com/kldzj/pzmod/internal/store"
)

// backups lists config snapshots and supports restore/delete/snapshot/diff,
// with type-to-filter.
type backups struct {
	entries []store.BackupEntry
	cursor  int
	loading bool
	filter  filterState
}

// NewBackups returns the backups screen.
func NewBackups() Screen { return &backups{loading: true} }

func (b *backups) Title() string { return "Backups" }

type backupsLoadedMsg struct {
	entries []store.BackupEntry
	err     error
}

func (b *backups) Init(s *Session) tea.Cmd { return b.load(s) }

func (b *backups) load(s *Session) tea.Cmd {
	id := s.Profile.ID
	return s.Do(func(ctx context.Context) tea.Msg {
		entries, err := s.Store.Backups(id)
		return backupsLoadedMsg{entries: entries, err: err}
	})
}

func (b *backups) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case backupsLoadedMsg:
		b.loading = false
		if msg.err != nil {
			return b, Fail(msg.err)
		}
		b.entries = msg.entries
		b.clamp()
		return b, nil
	case reloadBackupsMsg:
		return b, b.load(s)
	case restoredMsg:
		return b, tea.Batch(Toast("restored"), b.load(s))
	case tea.KeyMsg:
		if b.filter.active {
			if b.filter.handleKey(msg) {
				b.clamp()
				return b, nil
			}
		}
		switch msg.String() {
		case "esc":
			if b.filter.has() {
				b.filter.clear()
				b.clamp()
				return b, nil
			}
			return b, Pop()
		case "/":
			b.filter.start()
			return b, nil
		case "up", "k":
			if b.cursor > 0 {
				b.cursor--
			}
		case "down", "j":
			if b.cursor < len(b.shown())-1 {
				b.cursor++
			}
		case "pgup":
			bh := max(3, s.BodyHeight()-6)
			b.cursor = max(0, b.cursor-bh)
		case "pgdown":
			bh := max(3, s.BodyHeight()-6)
			if n := len(b.shown()); n > 0 {
				b.cursor = min(n-1, b.cursor+bh)
			}
		case "home":
			b.cursor = 0
		case "end":
			if n := len(b.shown()); n > 0 {
				b.cursor = n - 1
			}
		case "s":
			return b, b.snapshot(s)
		case "d":
			if e, ok := b.current(); ok {
				return b, Confirm("Delete backup "+e.ID+"?", b.deleteCmd(s, e.ID))
			}
		case "enter", "v":
			if e, ok := b.current(); ok {
				return b, Push(NewBackupDiff(s.Profile.ID, e.ID, s.Profile.IniPath))
			}
		case "r":
			if e, ok := b.current(); ok {
				return b, Confirm("Restore "+e.ID+"? (current config is backed up first)", b.restoreCmd(s, e.ID))
			}
		}
	}
	return b, nil
}

// shown returns the entries matching the current filter.
func (b *backups) shown() []store.BackupEntry {
	var out []store.BackupEntry
	for _, e := range b.entries {
		if filterMatch(b.filter.query, e.Note, e.Kind, e.ID, e.Timestamp) {
			out = append(out, e)
		}
	}
	return out
}

func (b *backups) clamp() {
	if n := len(b.shown()); b.cursor >= n {
		b.cursor = max(0, n-1)
	}
	if b.cursor < 0 {
		b.cursor = 0
	}
}

func (b *backups) current() (store.BackupEntry, bool) {
	sh := b.shown()
	if b.cursor < 0 || b.cursor >= len(sh) {
		return store.BackupEntry{}, false
	}
	return sh[b.cursor], true
}

func (b *backups) snapshot(s *Session) tea.Cmd {
	profile := *s.Profile
	return func() tea.Msg {
		if _, err := s.Svc.SnapshotProfile(profile, "manual snapshot", "manual"); err != nil {
			return ErrMsg{Err: err}
		}
		return reloadBackupsMsg{}
	}
}

func (b *backups) deleteCmd(s *Session, id string) tea.Cmd {
	pid := s.Profile.ID
	return func() tea.Msg {
		if err := s.Store.DeleteBackup(pid, id); err != nil {
			return ErrMsg{Err: err}
		}
		return reloadBackupsMsg{}
	}
}

func (b *backups) restoreCmd(s *Session, id string) tea.Cmd {
	pid := s.Profile.ID
	path := s.Profile.IniPath
	return func() tea.Msg {
		if err := s.Store.Restore(pid, id, path); err != nil {
			return ErrMsg{Err: err}
		}
		cfg, err := serverconfig.Load(path)
		if err != nil {
			return ErrMsg{Err: err}
		}
		s.Cfg = cfg
		return restoredMsg{}
	}
}

type reloadBackupsMsg struct{}
type restoredMsg struct{}

func (b *backups) View(s *Session) string {
	th := s.Theme
	if b.loading {
		return pad(th.Muted.Render("loading backups…"))
	}
	var sb strings.Builder

	if line := b.filter.view(th); line != "" {
		sb.WriteString(line + "\n\n")
	}

	sh := b.shown()
	if len(sh) == 0 {
		if b.filter.has() {
			sb.WriteString(th.Muted.Render("no backups match") + "\n\n")
		} else {
			sb.WriteString(th.Muted.Render("no backups yet") + "\n\n")
		}
	}
	bh := max(3, s.BodyHeight()-6)
	bStart, bEnd := listWindow(b.cursor, len(sh), bh)
	if bStart > 0 {
		sb.WriteString(th.Muted.Render(fmt.Sprintf("  ↑ %d more", bStart)) + "\n")
	}
	for i := bStart; i < bEnd; i++ {
		e := sh[i]
		sel := i == b.cursor
		left := relTimeRFC(e.Timestamp)
		if e.Note != "" {
			left += " - " + e.Note
		}
		right := metaLine("["+e.Kind+"]", humanize.Bytes(uint64(e.Size)))
		sb.WriteString(renderRow(th, s.ContentWidth(), cursorPrefix(th, sel), left, right, sel) + "\n")
	}
	if bEnd < len(sh) {
		sb.WriteString(th.Muted.Render(fmt.Sprintf("  ↓ %d more", len(sh)-bEnd)) + "\n")
	}

	sb.WriteString("\n" + th.Muted.Render("enter/v: diff   r: restore   d: delete   s: snapshot   /: filter   esc: back"))
	return pad(sb.String())
}
