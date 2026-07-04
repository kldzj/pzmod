package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/pkg/domain"
)

type listTab int

const (
	tabMods listTab = iota
	tabMaps
)

// modlist manages the ordered Mods= and Map= lines across two tabs.
type modlist struct {
	tab     listTab
	mods    []string
	maps    []string
	cursor  int
	grabbed bool

	modSuggest *domain.OrderPlan
	moved      map[string]bool
	previewing bool
	filter     filterState
}

// NewModList and NewLoadOrder both open the tabbed load-order screen.
func NewModList() Screen   { return &modlist{} }
func NewLoadOrder() Screen { return &modlist{} }

func (m *modlist) Title() string { return "Load order" }

type loadOrderMsg struct {
	plan domain.OrderPlan
	err  error
}

func (m *modlist) Init(s *Session) tea.Cmd {
	m.mods = append([]string(nil), s.Cfg.Mods()...)
	m.maps = append([]string(nil), s.Cfg.Maps()...)
	sm := s.Cfg.ServerMods()
	profile := *s.Profile
	return s.Do(func(ctx context.Context) tea.Msg {
		plan, err := s.Svc.SuggestLoadOrder(ctx, sm, profile)
		return loadOrderMsg{plan: plan, err: err}
	})
}

func (m *modlist) list() []string {
	if m.tab == tabMaps {
		return m.maps
	}
	return m.mods
}

func (m *modlist) setList(s *Session, v []string) {
	if m.tab == tabMaps {
		m.maps = v
		s.Cfg.SetMaps(v)
	} else {
		m.mods = v
		s.Cfg.SetMods(v)
	}
}

func (m *modlist) suggestion() *domain.OrderPlan {
	if m.tab == tabMaps {
		mp := domain.SuggestMapOrder(m.maps)
		return &mp
	}
	return m.modSuggest
}

// shown returns the active list filtered by the current query (locate mode).
func (m *modlist) shown() []string {
	l := m.list()
	if !m.filter.has() {
		return l
	}
	var out []string
	for _, it := range l {
		if filterMatch(m.filter.query, it, domain.ParseModRef(it).ID) {
			out = append(out, it)
		}
	}
	return out
}

// fullIndex returns the index of v in the active full list, or -1.
func (m *modlist) fullIndex(v string) int {
	for i, it := range m.list() {
		if it == v {
			return i
		}
	}
	return -1
}

func (m *modlist) clampCursor() {
	n := len(m.shown())
	if m.cursor >= n {
		m.cursor = max(0, n-1)
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// exitFilter clears the filter and repositions the cursor onto the currently
// selected item's index in the full list (so it is ready to reorder).
func (m *modlist) exitFilter() {
	sh := m.shown()
	target := 0
	if m.cursor >= 0 && m.cursor < len(sh) {
		if fi := m.fullIndex(sh[m.cursor]); fi >= 0 {
			target = fi
		}
	}
	m.filter.clear()
	m.cursor = target
}

func (m *modlist) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case loadOrderMsg:
		if msg.err == nil {
			p := msg.plan
			m.modSuggest = &p
		}
		return m, nil
	case tea.KeyMsg:
		if m.previewing {
			return m.updatePreview(s, msg)
		}
		// While typing a filter, enter/esc finish, navigation moves the filtered
		// cursor, and every other key (incl. letters that are reorder shortcuts in
		// normal mode) is typed into the query.
		if m.filter.active {
			switch msg.String() {
			case "enter", "esc":
				m.exitFilter()
				return m, nil
			case "up", "down", "pgup", "pgdown", "home", "end":
				// arrow/page keys navigate the narrowed list; fall through.
				// j/k are intentionally NOT here - they must type into the query,
				// matching the other filter screens.
			default:
				if m.filter.handleKey(msg) {
					m.clampCursor()
					return m, nil
				}
			}
		}
		switch msg.String() {
		case "esc":
			return m, Pop()
		case "/":
			m.filter.start()
			m.cursor, m.grabbed = 0, false
			return m, nil
		case "left", "right", "tab":
			m.filter.clear()
			m.tab = 1 - m.tab
			m.cursor, m.grabbed = 0, false
			return m, nil
		case "up", "k":
			if m.filter.has() {
				if m.cursor > 0 {
					m.cursor--
				}
			} else {
				m.moveCursor(s, -1)
			}
		case "down", "j":
			if m.filter.has() {
				if m.cursor < len(m.shown())-1 {
					m.cursor++
				}
			} else {
				m.moveCursor(s, +1)
			}
		case "pgup":
			h := max(3, s.BodyHeight()-6)
			m.cursor = clamp(m.cursor-h, 0, max(0, len(m.shown())-1))
		case "pgdown":
			h := max(3, s.BodyHeight()-6)
			m.cursor = clamp(m.cursor+h, 0, max(0, len(m.shown())-1))
		case "home":
			if len(m.shown()) > 0 {
				m.cursor = 0
			}
		case "end":
			if n := len(m.shown()); n > 0 {
				m.cursor = n - 1
			}
		case "K":
			m.shift(s, -1)
		case "J":
			m.shift(s, +1)
		case "t":
			m.moveTo(s, 0)
		case "b":
			m.moveTo(s, len(m.list())-1)
		case " ", "g":
			m.grabbed = !m.grabbed
		case "s":
			if sg := m.suggestion(); sg != nil && len(sg.Moved) > 0 {
				m.moved = map[string]bool{}
				for _, mv := range sg.Moved {
					m.moved[mv] = true
				}
				m.previewing = true
			} else {
				return m, Toast("order already looks optimal")
			}
		}
	}
	return m, nil
}

func (m *modlist) updatePreview(s *Session, msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		m.setList(s, append([]string(nil), m.suggestion().Ordered...))
		m.previewing = false
		return m, Toast("order applied (unsaved)")
	case "n", "esc":
		m.previewing = false
	}
	return m, nil
}

func (m *modlist) moveCursor(s *Session, d int) {
	if len(m.list()) == 0 {
		return
	}
	if m.grabbed {
		m.shift(s, d)
		return
	}
	m.cursor = clamp(m.cursor+d, 0, len(m.list())-1)
}

func (m *modlist) shift(s *Session, d int) {
	l := m.list()
	j := m.cursor + d
	if j < 0 || j >= len(l) {
		return
	}
	l[m.cursor], l[j] = l[j], l[m.cursor]
	m.cursor = j
	m.setList(s, l)
}

func (m *modlist) moveTo(s *Session, to int) {
	l := m.list()
	if m.cursor == to || to < 0 || to >= len(l) {
		return
	}
	item := l[m.cursor]
	l = append(l[:m.cursor], l[m.cursor+1:]...)
	out := make([]string, 0, len(l)+1)
	out = append(out, l[:to]...)
	out = append(out, item)
	out = append(out, l[to:]...)
	m.cursor = to
	m.setList(s, out)
}

func (m *modlist) View(s *Session) string {
	th := s.Theme
	if m.previewing {
		return m.viewPreview(s)
	}
	var b strings.Builder
	b.WriteString(tabHeader(th, m.tab) + "\n\n")
	if line := m.filter.view(th); line != "" {
		b.WriteString(line + "\n\n")
	}

	display := m.shown()
	if len(m.list()) == 0 {
		b.WriteString(th.Muted.Render("empty - add from Search Workshop") + "\n\n")
	} else if len(display) == 0 {
		b.WriteString(th.Muted.Render(fmt.Sprintf("no matches for %q", m.filter.query)) + "\n\n")
	}
	lh := max(3, s.BodyHeight()-6-m.filter.chrome())
	lStart, lEnd := listWindow(m.cursor, len(display), lh)
	if lStart > 0 {
		b.WriteString(th.Muted.Render(fmt.Sprintf("  ↑ %d more", lStart)) + "\n")
	}
	for i := lStart; i < lEnd; i++ {
		item := display[i]
		sel := i == m.cursor
		ord := i + 1
		if m.filter.has() {
			ord = m.fullIndex(item) + 1
		}
		ref := domain.ParseModRef(item)
		label := ref.ID
		if label == "" {
			label = item
		}
		prefix := cursorPrefix(th, sel) + th.Muted.Render(fmt.Sprintf("%2d. ", ord))
		right := ""
		switch {
		case sel && m.grabbed && !m.filter.has():
			right = "✥ moving"
		case ref.Workshop != "":
			right = "pinned " + ref.Workshop
		}
		b.WriteString(renderRow(th, s.ContentWidth(), prefix, label, right, sel) + "\n")
	}
	if lEnd < len(display) {
		b.WriteString(th.Muted.Render(fmt.Sprintf("  ↓ %d more", len(display)-lEnd)) + "\n")
	}
	b.WriteString("\n")
	var hint string
	switch {
	case m.filter.has():
		hint = "locate: ↑/↓ move · enter jump to item · esc clear filter"
	case m.grabbed:
		hint = th.Warn.Render("MOVING") + " - ↑/↓ to position, space to drop"
	default:
		hint = "←/→: switch tab   space: grab   J/K: move   t/b: top/bottom   s: suggest   /: filter   esc: back"
		if sg := m.suggestion(); sg != nil && len(sg.Moved) > 0 {
			hint = "←/→: switch tab   space: grab   J/K: move   t/b: top/bottom   s: suggest ✦   /: filter   esc: back"
		}
	}
	b.WriteString(th.Muted.Render(hint))
	return pad(b.String())
}

func tabHeader(th Theme, tab listTab) string {
	mods, maps := " Mods ", " Maps "
	if tab == tabMods {
		mods = th.SelectedItem.Render(mods)
		maps = th.Muted.Render(maps)
	} else {
		maps = th.SelectedItem.Render(maps)
		mods = th.Muted.Render(mods)
	}
	return mods + maps
}

func (m *modlist) viewPreview(s *Session) string {
	th := s.Theme
	sg := m.suggestion()
	var b strings.Builder
	b.WriteString(th.Subtitle.Render("Suggested order") + "\n\n")
	for i, item := range sg.Ordered {
		mark := "  "
		if m.moved[item] {
			mark = th.OK.Render("→ ")
		}
		reason := ""
		if r, ok := sg.Reasons[item]; ok && r != "" {
			reason = "  " + th.Muted.Render(r)
		}
		label := domain.ParseModRef(item).ID
		if label == "" {
			label = item
		}
		b.WriteString(fmt.Sprintf("%2d. %s%s%s\n", i+1, mark, label, reason))
	}
	if len(sg.Cycles) > 0 {
		b.WriteString("\n" + th.Warn.Render(fmt.Sprintf("%d dependency cycle(s) - those mods kept their order", len(sg.Cycles))))
	}
	b.WriteString("\n\n" + th.Muted.Render("enter/y: apply   esc/n: cancel"))
	return pad(b.String())
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
