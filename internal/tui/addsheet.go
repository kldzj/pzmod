package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/internal/build"
	"github.com/kldzj/pzmod/internal/domain"
	"github.com/kldzj/pzmod/internal/steam"
)

type addRow struct {
	label string
	value string
	isMap bool
	on    bool
}

// addSheet lets the user choose which declared mods/maps an item contributes.
type addSheet struct {
	id     string
	title  string
	rows   []addRow
	cursor int
}

// NewAddSheet builds the sheet from a content item's declared mods/maps.
func NewAddSheet(item steam.WorkshopItem) Screen {
	parsed := item.Parse()
	var rows []addRow
	for _, m := range parsed.Mods {
		rows = append(rows, addRow{label: m, value: m, isMap: false, on: true})
	}
	for _, mp := range parsed.Maps {
		rows = append(rows, addRow{label: mp, value: mp, isMap: true, on: true})
	}
	return newAddSheetWith(item.PublishedFileID, itemTitle(&item), rows)
}

func newAddSheetWith(id, title string, rows []addRow) *addSheet {
	return &addSheet{id: id, title: title, rows: rows}
}

func (a *addSheet) Title() string { return "Add to server" }

func (a *addSheet) Init(s *Session) tea.Cmd { return nil }

func (a *addSheet) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	switch key.String() {
	case "esc":
		return a, Pop()
	case "up", "k":
		if a.cursor > 0 {
			a.cursor--
		}
	case "down", "j":
		if a.cursor < len(a.rows)-1 {
			a.cursor++
		}
	case " ":
		if a.cursor < len(a.rows) {
			a.rows[a.cursor].on = !a.rows[a.cursor].on
		}
	case "enter", "a":
		return a, a.apply(s)
	}
	return a, nil
}

func (a *addSheet) apply(s *Session) tea.Cmd {
	sm := s.Cfg.ServerMods().AddItem(a.id)
	explicit := s.Build() == build.B42
	for _, r := range a.rows {
		if !r.on {
			continue
		}
		if r.isMap {
			sm = sm.AddMap(r.value)
		} else {
			sm = sm.AddMod(domain.FormatModRef(a.id, r.value, explicit))
		}
	}
	s.Cfg.ApplyServerMods(sm)
	return tea.Batch(Pop(), func() tea.Msg { return modsChangedMsg{toast: "added " + a.title + " (unsaved)"} })
}

func (a *addSheet) View(s *Session) string {
	th := s.Theme
	var b strings.Builder
	b.WriteString(th.Subtitle.Render("Add \""+a.title+"\"?") + "\n\n")
	mapsSeen := false
	for i, r := range a.rows {
		if r.isMap && !mapsSeen {
			b.WriteString("\n" + th.Muted.Render("Maps") + "\n")
			mapsSeen = true
		} else if i == 0 {
			b.WriteString(th.Muted.Render("Mods") + "\n")
		}
		box := "[ ] "
		if r.on {
			box = th.OK.Render("[x] ")
		}
		right := ""
		if r.isMap {
			right = "adds to rotation"
		}
		b.WriteString(renderRow(th, s.ContentWidth(), cursorPrefix(th, i == a.cursor)+box, r.label, right, i == a.cursor) + "\n")
	}
	b.WriteString("\n" + th.Muted.Render("space: toggle   enter: add   esc: cancel"))
	return pad(b.String())
}
