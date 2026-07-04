package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/internal/openurl"
	"github.com/kldzj/pzmod/pkg/steam"
)

type installedRow struct {
	id    string
	title string
	size  uint64
	mods  int
	maps  int
	ok    bool // details fetched
}

// installed is the item-centric view of what the profile has added.
type installed struct {
	rows    []installedRow
	decl    map[string]domain.ModDecl
	cursor  int
	loading bool
	load    loader
	filter  filterState
}

// NewInstalled returns the Installed Mods screen.
func NewInstalled() Screen { return &installed{loading: true, load: newLoader()} }

func (in *installed) Title() string { return "Installed Mods" }

type installedLoadedMsg struct {
	items []steam.WorkshopItem
	err   error
}

func (in *installed) Init(s *Session) tea.Cmd {
	return tea.Batch(in.load.tick(), in.reload(s))
}

func (in *installed) reload(s *Session) tea.Cmd {
	ids := append([]string(nil), s.Cfg.WorkshopItems()...)
	return s.Do(func(ctx context.Context) tea.Msg {
		items, _, err := s.Svc.Details(ctx, ids)
		return installedLoadedMsg{items: items, err: err}
	})
}

func (in *installed) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	if cmd, ok := in.load.update(msg); ok {
		if in.loading {
			return in, cmd
		}
		return in, nil
	}
	switch msg := msg.(type) {
	case installedLoadedMsg:
		in.loading = false
		if msg.err != nil {
			return in, Fail(msg.err)
		}
		in.build(s, msg.items)
		return in, nil
	// Both modsChangedMsg and resumedMsg can fire on a single add; the reload is idempotent.
	case modsChangedMsg:
		return in, in.reload(s)
	case resumedMsg:
		return in, in.reload(s)
	case tea.KeyMsg:
		if in.filter.active {
			if in.filter.handleKey(msg) {
				in.clampCursor()
				return in, nil
			}
		}
		switch msg.String() {
		case "esc":
			if in.filter.has() {
				in.filter.clear()
				in.clampCursor()
				return in, nil
			}
			return in, Pop()
		case "/":
			in.filter.start()
			return in, nil
		case "up", "k":
			if in.cursor > 0 {
				in.cursor--
			}
		case "down", "j":
			if in.cursor < len(in.shown())-1 {
				in.cursor++
			}
		case "pgup":
			h := max(3, s.BodyHeight()-4)
			in.cursor = max(0, in.cursor-h)
		case "pgdown":
			h := max(3, s.BodyHeight()-4)
			in.cursor = min(max(0, len(in.shown())-1), in.cursor+h)
		case "home":
			in.cursor = 0
		case "end":
			in.cursor = max(0, len(in.shown())-1)
		case "enter":
			if r, ok := in.current(); ok {
				return in, Push(NewDetail(r.id))
			}
		case "a":
			return in, Push(NewAddByID())
		case "x":
			if r, ok := in.current(); ok {
				return in, in.confirmRemove(s, r)
			}
		case "o":
			if r, ok := in.current(); ok {
				tmp := steam.WorkshopItem{PublishedFileID: r.id}
				_ = openurl.Open(tmp.WorkshopURL())
				return in, Toast("opening in browser…")
			}
		}
	}
	return in, nil
}

func (in *installed) build(s *Session, items []steam.WorkshopItem) {
	byID := map[string]steam.WorkshopItem{}
	for _, it := range items {
		byID[it.PublishedFileID] = it
	}
	in.decl = map[string]domain.ModDecl{}
	in.rows = nil
	for _, id := range s.Cfg.WorkshopItems() {
		it, ok := byID[id]
		if !ok {
			in.rows = append(in.rows, installedRow{id: id, title: id, ok: false})
			continue
		}
		p := it.Parse()
		in.decl[id] = domain.ModDecl{Mods: p.Mods, Maps: p.Maps}
		in.rows = append(in.rows, installedRow{
			id: id, title: itemTitle(&it), size: uint64(it.FileSize),
			mods: len(p.Mods), maps: len(p.Maps), ok: true,
		})
	}
	in.clampCursor()
}

// shown returns the rows matching the current filter (title, workshop ID, and
// declared mod IDs / map names).
func (in *installed) shown() []installedRow {
	if !in.filter.has() {
		return in.rows
	}
	var out []installedRow
	for _, r := range in.rows {
		fields := []string{r.title, r.id}
		if d, ok := in.decl[r.id]; ok {
			fields = append(fields, d.Mods...)
			fields = append(fields, d.Maps...)
		}
		if filterMatch(in.filter.query, fields...) {
			out = append(out, r)
		}
	}
	return out
}

func (in *installed) clampCursor() {
	if n := len(in.shown()); in.cursor >= n {
		in.cursor = max(0, n-1)
	}
	if in.cursor < 0 {
		in.cursor = 0
	}
}

func (in *installed) current() (installedRow, bool) {
	sh := in.shown()
	if in.cursor < 0 || in.cursor >= len(sh) {
		return installedRow{}, false
	}
	return sh[in.cursor], true
}

func (in *installed) confirmRemove(s *Session, r installedRow) tea.Cmd {
	plan := domain.PlanRemoval(r.id, in.decl, s.Cfg.ServerMods())
	desc := "item " + r.id
	if len(plan.Mods) > 0 {
		desc += " + mod " + strings.Join(plan.Mods, ", ")
	}
	if len(plan.Maps) > 0 {
		desc += " + map " + strings.Join(plan.Maps, ", ")
	}
	return Confirm("Remove "+r.title+"? ("+desc+")", func() tea.Msg {
		s.Cfg.ApplyServerMods(plan.Apply(s.Cfg.ServerMods()))
		return modsChangedMsg{toast: "removed " + r.title}
	})
}

func (in *installed) View(s *Session) string {
	th := s.Theme
	if in.loading {
		return pad(in.load.view(th, "loading installed items…"))
	}
	var b strings.Builder
	if line := in.filter.view(th); line != "" {
		b.WriteString(line + "\n\n")
	}
	rows := in.shown()
	total := len(rows)
	if len(in.rows) == 0 {
		b.WriteString(th.Muted.Render("nothing installed yet - press a to add by ID, or use Search Workshop") + "\n\n")
	} else if total == 0 {
		b.WriteString(th.Muted.Render(fmt.Sprintf("no matches for %q", in.filter.query)) + "\n\n")
	}
	h := max(3, s.BodyHeight()-4-in.filter.chrome())
	start, end := listWindow(in.cursor, total, h)
	if start > 0 {
		b.WriteString(th.Muted.Render(fmt.Sprintf("  ↑ %d more", start)) + "\n")
	}
	for i := start; i < end; i++ {
		r := rows[i]
		sel := i == in.cursor
		right := ""
		if r.ok {
			right = metaLine(humanize.Bytes(r.size), modsMapsLabel(r.mods, r.maps))
		} else {
			right = th.Warn.Render("unavailable")
		}
		b.WriteString(renderRow(th, s.ContentWidth(), cursorPrefix(th, sel), r.title, right, sel) + "\n")
	}
	if end < total {
		b.WriteString(th.Muted.Render(fmt.Sprintf("  ↓ %d more", total-end)) + "\n")
	}
	b.WriteString("\n" + th.Muted.Render("enter: details   a: add by ID   x: remove   o: open   /: filter   esc: back"))
	return pad(b.String())
}

func modsMapsLabel(mods, maps int) string {
	parts := []string{fmt.Sprintf("%d mods", mods)}
	if maps > 0 {
		parts = append(parts, fmt.Sprintf("%d maps", maps))
	}
	return strings.Join(parts, " · ")
}
