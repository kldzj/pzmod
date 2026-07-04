package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/kldzj/pzmod/pkg/build"
	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/internal/openurl"
	"github.com/kldzj/pzmod/pkg/service"
	"github.com/kldzj/pzmod/pkg/steam"
)

// deps resolves the dependency closure of some seed items and lets the user
// pick which additions to apply.
type deps struct {
	seeds   []string
	loading bool
	load    loader
	plan    service.ResolvePlan
	rows    []depRow
	cursor  int
}

// depRow is a single selectable row in the deps list.
type depRow struct {
	id       string
	title    string
	mods     []string // declared mod IDs (one item may declare several)
	size     uint64
	selected bool
	seed     bool
}

// NewDeps returns the dependency-resolution screen for the given seed IDs.
func NewDeps(seeds []string) Screen { return &deps{seeds: seeds, loading: true, load: newLoader()} }

func (d *deps) Title() string { return "Resolve dependencies" }

type depsResolvedMsg struct {
	plan service.ResolvePlan
	err  error
}

func (d *deps) Init(s *Session) tea.Cmd {
	seeds := d.seeds
	installed := s.Cfg.ServerMods()
	return tea.Batch(d.load.tick(), s.Do(func(ctx context.Context) tea.Msg {
		plan, err := s.Svc.Resolve(ctx, seeds, installed)
		return depsResolvedMsg{plan: plan, err: err}
	}))
}

// footerLines returns the number of lines the footer section occupies below the
// row window, so the window height can be computed to leave exactly enough room.
func (d *deps) footerLines() int {
	n := 1 // blank separator
	n += 1 // selected total
	n += 1 // hint line
	n += 2 // scroll indicators (reserve both even when only one or zero shown)
	if len(d.plan.Missing) > 0 {
		n++ // +1 for the collapsed unavailable summary
	}
	if len(d.plan.MultiMod) > 0 {
		n++
	}
	if len(d.plan.Cycles) > 0 {
		n++
	}
	return n
}

func (d *deps) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	if cmd, ok := d.load.update(msg); ok {
		if d.loading {
			return d, cmd
		}
		return d, nil
	}
	switch msg := msg.(type) {
	case depsResolvedMsg:
		if msg.err != nil {
			return d, tea.Batch(Fail(msg.err), Pop())
		}
		d.loading = false
		d.plan = msg.plan
		d.buildRows()
		return d, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return d, Pop()
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			if d.cursor < len(d.rows)-1 {
				d.cursor++
			}
		case "pgup":
			h := max(3, s.BodyHeight()-d.footerLines())
			d.cursor = max(0, d.cursor-h)
		case "pgdown":
			h := max(3, s.BodyHeight()-d.footerLines())
			if n := len(d.rows); n > 0 {
				d.cursor = min(n-1, d.cursor+h)
			}
		case "home":
			d.cursor = 0
		case "end":
			if n := len(d.rows); n > 0 {
				d.cursor = n - 1
			}
		case " ":
			if d.cursor < len(d.rows) {
				d.rows[d.cursor].selected = !d.rows[d.cursor].selected
			}
		case "a", "enter":
			return d, d.apply(s)
		case "o":
			if d.cursor < len(d.rows) {
				tmp := steam.WorkshopItem{PublishedFileID: d.rows[d.cursor].id}
				_ = openurl.Open(tmp.WorkshopURL())
				return d, Toast("opening in browser…")
			}
		case "u":
			if len(d.plan.Missing) > 0 {
				title := fmt.Sprintf("Unavailable items (%d)", len(d.plan.Missing))
				return d, Push(NewInfoList(title, d.plan.Missing))
			}
		}
	}
	return d, nil
}

func (d *deps) buildRows() {
	seedSet := map[string]bool{}
	for _, s := range d.seeds {
		seedSet[s] = true
	}
	d.rows = nil
	for _, id := range d.plan.AddWorkshopItems {
		item := d.plan.Items[id]
		parsed := item.Parse()
		d.rows = append(d.rows, depRow{
			id:       id,
			title:    itemTitleOr(item, id),
			mods:     parsed.Mods,
			size:     uint64(item.FileSize),
			selected: true,
			seed:     seedSet[id],
		})
	}
}

func (d *deps) apply(s *Session) tea.Cmd {
	sm := s.Cfg.ServerMods()
	n := 0
	for _, r := range d.rows {
		if !r.selected {
			continue
		}
		item := d.plan.Items[r.id]
		sm = sm.AddItem(r.id)
		parsed := item.Parse()
		explicit := s.Build() == build.B42
		for _, m := range parsed.Mods {
			sm = sm.AddMod(domain.FormatModRef(r.id, m, explicit))
		}
		for _, mp := range parsed.Maps {
			sm = sm.AddMap(mp)
		}
		n++
	}
	s.Cfg.ApplyServerMods(sm)
	return tea.Batch(Toast(fmt.Sprintf("added %d item(s) (unsaved)", n)), Pop())
}

func (d *deps) View(s *Session) string {
	th := s.Theme
	if d.loading {
		return pad(d.load.view(th, "resolving dependencies…"))
	}

	var b strings.Builder
	if len(d.rows) == 0 && len(d.plan.Missing) == 0 {
		b.WriteString(th.OK.Render("nothing to add - dependencies already satisfied") + "\n")
		b.WriteString(th.Muted.Render("esc: back"))
		return pad(b.String())
	}

	var total uint64
	for _, r := range d.rows {
		if r.selected {
			total += r.size
		}
	}
	rowCount := len(d.rows)
	// Reserve exact footer height so "selected total" is never pushed off by the scroll indicator.
	h := max(3, s.BodyHeight()-d.footerLines())
	wStart, wEnd := listWindow(d.cursor, rowCount, h)
	if wStart > 0 {
		b.WriteString(th.Muted.Render(fmt.Sprintf("  ↑ %d more", wStart)) + "\n")
	}
	for i := wStart; i < wEnd; i++ {
		r := d.rows[i]
		sel := i == d.cursor
		box := "[ ] "
		if r.selected {
			box = th.OK.Render("[x] ")
		}
		tag := "dependency"
		if r.seed {
			tag = "requested"
		}
		// Append the declared mod IDs so clashing IDs are distinguishable.
		modTag := strings.Join(r.mods, "+")
		right := metaLine(humanize.Bytes(r.size), modTag, tag)
		b.WriteString(renderRow(th, s.ContentWidth(), box, r.title, right, sel) + "\n")
	}
	if wEnd < rowCount {
		b.WriteString(th.Muted.Render(fmt.Sprintf("  ↓ %d more", rowCount-wEnd)) + "\n")
	}

	// Collapse unavailable items to one summary line (avoids eating list height).
	if len(d.plan.Missing) > 0 {
		b.WriteString(th.Warn.Render(fmt.Sprintf("  ⚠ %d unavailable (delisted/private) - not added", len(d.plan.Missing))) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(th.Muted.Render(fmt.Sprintf("selected total: %s", humanize.Bytes(total))) + "\n")
	if len(d.plan.MultiMod) > 0 {
		b.WriteString(th.Warn.Render(fmt.Sprintf("%d item(s) declare multiple mod IDs", len(d.plan.MultiMod))) + "\n")
	}
	if len(d.plan.Cycles) > 0 {
		b.WriteString(th.Warn.Render(fmt.Sprintf("%d dependency cycle(s) detected", len(d.plan.Cycles))) + "\n")
	}
	hint := "space: toggle   a/enter: apply   o: open   esc: back"
	if len(d.plan.Missing) > 0 {
		hint += "   u: view unavailable"
	}
	b.WriteString(th.Muted.Render(hint))
	return pad(b.String())
}

func itemTitleOr(it steam.WorkshopItem, id string) string {
	if it.Title != "" {
		return it.Title
	}
	return id
}
