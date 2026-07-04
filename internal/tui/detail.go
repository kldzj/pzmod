package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kldzj/pzmod/internal/bbcode"
	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/internal/openurl"
	"github.com/kldzj/pzmod/pkg/steam"
)

// detail shows a single Workshop item with a scrollable description and the
// actions to add it (and, from M8, resolve its dependencies).
type detail struct {
	id      string
	item    *steam.WorkshopItem
	parsed  *steam.ParsedItem
	vp      viewport.Model
	load    loader
	loading bool
	ready   bool
}

// NewDetail returns the item detail screen for a Workshop ID.
func NewDetail(id string) Screen { return &detail{id: id, loading: true, load: newLoader()} }

func (d *detail) Title() string { return "Mod details" }

type detailLoadedMsg struct {
	item *steam.WorkshopItem
	err  error
}

type detailRemovePlanMsg struct {
	plan  domain.RemovalPlan
	title string
}

type detailRemovedMsg struct{ title string }

func (d *detail) Init(s *Session) tea.Cmd {
	id := d.id
	return tea.Batch(d.load.tick(), s.Do(func(ctx context.Context) tea.Msg {
		items, _, err := s.Svc.Details(ctx, []string{id})
		if err != nil {
			return detailLoadedMsg{err: err}
		}
		if len(items) == 0 {
			return detailLoadedMsg{err: fmt.Errorf("item %s not found", id)}
		}
		return detailLoadedMsg{item: &items[0]}
	}))
}

func (d *detail) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	if cmd, ok := d.load.update(msg); ok {
		if d.loading {
			return d, cmd
		}
		return d, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.resize(s)
		return d, nil
	case detailLoadedMsg:
		if msg.err != nil {
			return d, tea.Batch(Fail(msg.err), Pop())
		}
		d.item = msg.item
		d.parsed = msg.item.Parse()
		d.loading = false
		d.resize(s)
		return d, nil
	case detailRemovePlanMsg:
		desc := "item " + msg.plan.Item
		if len(msg.plan.Mods) > 0 {
			desc += " + mod " + strings.Join(msg.plan.Mods, ", ")
		}
		if len(msg.plan.Maps) > 0 {
			desc += " + map " + strings.Join(msg.plan.Maps, ", ")
		}
		plan := msg.plan
		title := msg.title
		return d, Confirm("Remove "+title+"? ("+desc+")", func() tea.Msg {
			s.Cfg.ApplyServerMods(plan.Apply(s.Cfg.ServerMods()))
			return detailRemovedMsg{title: title}
		})
	case detailRemovedMsg:
		// Emit ONLY modsChangedMsg - the modsChangedMsg case below Pop()s this
		// screen (a single pop). Batching an explicit Pop() here would race the
		// delegated modsChangedMsg's Pop() and double-pop the stack.
		return d, func() tea.Msg { return modsChangedMsg{toast: "removed " + msg.title} }
	case modsChangedMsg:
		// quickAdd took the instant-add path (single mod, no maps): close the
		// detail screen so the caller (search/installed) resumes. The toast has
		// already been shown by app.go before delegating here.
		return d, Pop()
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return d, Pop()
		case "a":
			return d, d.add(s)
		case "x":
			if d.item != nil && installedSet(s)[d.item.PublishedFileID] {
				return d, d.removeCmd(s)
			}
		case "o":
			if d.item != nil {
				_ = openurl.Open(d.item.WorkshopURL())
				return d, Toast("opening in browser…")
			}
		case "d":
			if d.item != nil {
				return d, Push(NewDeps([]string{d.item.PublishedFileID}))
			}
		}
	}
	if d.ready {
		var cmd tea.Cmd
		d.vp, cmd = d.vp.Update(msg)
		return d, cmd
	}
	return d, nil
}

func (d *detail) resize(s *Session) {
	w := max(20, s.Width-2)
	h := max(3, s.BodyHeight()-d.chromeHeight())
	d.vp = viewport.New(w, h)
	d.ready = true
	if d.item != nil {
		rendered := bbcode.Render(d.item.Description, bbStyles(s.Theme))
		body := lipgloss.NewStyle().Width(w).Render(rendered)
		d.vp.SetContent(body)
	}
}

// chromeHeight is the number of body lines used by everything except the
// scrollable viewport (header rows + blank line + footer hint), so the viewport
// is sized to leave the footer hint visible regardless of how tall the header
// grows for installed items (mods + maps rows).
func (d *detail) chromeHeight() int {
	// title + meta + hyperlink + blank line + hint.
	n := 5
	if d.parsed != nil {
		if len(d.parsed.Mods) > 0 {
			n++
		}
		if len(d.parsed.Maps) > 0 {
			n++
		}
	}
	if d.item != nil && len(tagLabels(d.item.Tags)) > 0 {
		n++
	}
	return n
}

func (d *detail) add(s *Session) tea.Cmd {
	if d.item == nil {
		return nil
	}
	// quickAdd handles collections (routes to dependency resolution) and content
	// items (adds the item + its declared mods/maps). replaceTop=true makes
	// sheet/deps replace the detail screen atomically (ReplaceMsg), avoiding the
	// Pop()+PushMsg race. The instant-add path returns modsChangedMsg, which is
	// handled above to Pop this screen.
	return quickAdd(s, d.item.PublishedFileID, true)
}

// removeCmd fetches the full declarations for all installed items so the
// removal plan can avoid dropping mods/maps that another item also claims.
func (d *detail) removeCmd(s *Session) tea.Cmd {
	id := d.item.PublishedFileID
	ids := append([]string(nil), s.Cfg.WorkshopItems()...)
	title := itemTitle(d.item)
	return s.Do(func(ctx context.Context) tea.Msg {
		items, _, err := s.Svc.Details(ctx, ids)
		if err != nil && len(items) == 0 {
			return ErrMsg{Err: err} // can't safely compute the plan; abort the remove
		}
		decl := map[string]domain.ModDecl{}
		for _, it := range items {
			p := it.Parse()
			decl[it.PublishedFileID] = domain.ModDecl{Mods: p.Mods, Maps: p.Maps}
		}
		plan := domain.PlanRemoval(id, decl, s.Cfg.ServerMods())
		return detailRemovePlanMsg{plan: plan, title: title}
	})
}

func (d *detail) View(s *Session) string {
	th := s.Theme
	if d.loading {
		return pad(d.load.view(th, "loading mod details…"))
	}
	if d.item == nil {
		return pad(th.Error.Render("not found"))
	}

	var b strings.Builder
	b.WriteString(th.Title.Render(itemTitle(d.item)))
	if installedSet(s)[d.item.PublishedFileID] {
		b.WriteString("  " + th.OK.Render("✓ installed"))
	}
	b.WriteString("\n")

	meta := []string{humanize.Bytes(uint64(d.item.FileSize))}
	if d.item.TimeUpdated > 0 {
		meta = append(meta, "updated "+relTime(d.item.TimeUpdated))
	}
	if d.item.Subscriptions > 0 {
		meta = append(meta, humanize.Comma(d.item.Subscriptions)+" subscribers")
	}
	if d.item.IsCollection() {
		meta = append(meta, fmt.Sprintf("collection · %d items", len(d.item.Children)))
	} else if n := len(d.item.Children); n > 0 {
		meta = append(meta, depLabel(n))
	}
	b.WriteString(th.Muted.Render(metaLine(meta...)) + "\n")

	if len(d.parsed.Mods) > 0 {
		b.WriteString(th.Muted.Render("mods: ") + strings.Join(d.parsed.Mods, ", ") + "\n")
	}
	if len(d.parsed.Maps) > 0 {
		b.WriteString(th.Muted.Render("maps: ") + strings.Join(d.parsed.Maps, ", ") + "\n")
	}
	if tags := tagLabels(d.item.Tags); len(tags) > 0 {
		b.WriteString(chips(th, tags...) + "\n")
	}
	b.WriteString(th.Muted.Render(hyperlink(d.item.WorkshopURL(), "open in Steam ↗")) + "\n\n")

	b.WriteString(d.vp.View() + "\n")
	if installedSet(s)[d.item.PublishedFileID] {
		b.WriteString(th.Muted.Render("a: add   d: deps   x: remove   o: open   esc: back"))
	} else {
		b.WriteString(th.Muted.Render("a: add   d: deps   o: open   esc: back"))
	}
	return pad(b.String())
}

func depLabel(n int) string {
	if n == 1 {
		return "1 dependency"
	}
	return fmt.Sprintf("%d dependencies", n)
}

func tagLabels(tags []steam.WorkshopTag) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if t.Tag != "" {
			out = append(out, t.Tag)
		}
	}
	return out
}

func itemTitle(it *steam.WorkshopItem) string {
	if it.Title != "" {
		return it.Title
	}
	return it.PublishedFileID
}
