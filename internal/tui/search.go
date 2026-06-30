package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/kldzj/pzmod/internal/steam"
)

// search is the Workshop search screen: a live query input plus a results list,
// with debounced, generation-guarded async queries and cursor-based paging.
type search struct {
	input   textinput.Model
	spinner spinner.Model

	results     []steam.WorkshopItem
	cursor      int
	total       int
	nextCursor  string
	loading     bool
	loadingMore bool
	gen         int // bumped per query; stale results are discarded
}

// NewSearch returns the Workshop search screen.
func NewSearch() Screen {
	ti := textinput.New()
	ti.Placeholder = "type to search the Workshop…"
	ti.Focus()
	ti.Width = 40
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return &search{input: ti, spinner: sp}
}

func (sc *search) Title() string { return "Search Workshop" }

type debounceMsg struct{ gen int }

type searchResultMsg struct {
	gen        int
	page       steam.Page
	err        error
	appendMode bool
}

func (sc *search) Init(s *Session) tea.Cmd {
	sc.loading = true
	return tea.Batch(textinput.Blink, sc.spinner.Tick, sc.runSearch(s, sc.gen, "", false))
}

func (sc *search) runSearch(s *Session, gen int, cursor string, appendMode bool) tea.Cmd {
	q := sc.input.Value()
	return s.Do(func(ctx context.Context) tea.Msg {
		page, err := s.Svc.Search(ctx, steam.Query{SearchText: q, PerPage: 40, Cursor: cursor})
		return searchResultMsg{gen: gen, page: page, err: err, appendMode: appendMode}
	})
}

func (sc *search) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		sc.spinner, cmd = sc.spinner.Update(msg)
		return sc, cmd

	case debounceMsg:
		if msg.gen != sc.gen {
			return sc, nil // superseded by a newer keystroke
		}
		sc.loading = true
		return sc, sc.runSearch(s, sc.gen, "", false)

	case searchResultMsg:
		if msg.gen != sc.gen {
			return sc, nil // stale
		}
		sc.loading, sc.loadingMore = false, false
		if msg.err != nil {
			return sc, Fail(msg.err)
		}
		sc.total = msg.page.Total
		sc.nextCursor = msg.page.NextCursor
		if msg.appendMode {
			sc.results = append(sc.results, msg.page.Items...)
		} else {
			sc.results = msg.page.Items
			sc.cursor = 0
		}
		return sc, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return sc, Pop()
		case "up", "ctrl+p":
			if sc.cursor > 0 {
				sc.cursor--
			}
			return sc, nil
		case "down", "ctrl+n":
			if sc.cursor < len(sc.results)-1 {
				sc.cursor++
			}
			return sc, sc.maybeLoadMore(s)
		case "pgup":
			h := max(3, s.BodyHeight()-7)
			sc.cursor = max(0, sc.cursor-h)
			return sc, nil
		case "pgdown":
			h := max(3, s.BodyHeight()-7)
			if n := len(sc.results); n > 0 {
				sc.cursor = min(n-1, sc.cursor+h)
			}
			return sc, sc.maybeLoadMore(s)
		case "home":
			sc.cursor = 0
			return sc, nil
		case "end":
			if n := len(sc.results); n > 0 {
				sc.cursor = n - 1
			}
			return sc, sc.maybeLoadMore(s)
		case "enter":
			if item := sc.current(); item != nil {
				return sc, Push(NewDetail(item.PublishedFileID)) // open detail
			}
			return sc, nil
		case "ctrl+o":
			id := parseWorkshopID(sc.input.Value())
			if id == "" {
				return sc, Toast("type a 6+ digit Workshop ID, then ctrl+o to add it")
			}
			return sc, quickAdd(s, id, false)
		}
		// Otherwise treat as text input and (re)trigger a debounced search.
		prev := sc.input.Value()
		var cmd tea.Cmd
		sc.input, cmd = sc.input.Update(msg)
		if sc.input.Value() != prev {
			sc.gen++
			return sc, tea.Batch(cmd, debounce(sc.gen))
		}
		return sc, cmd
	}
	return sc, nil
}

// maybeLoadMore fetches the next page when the cursor reaches the end.
func (sc *search) maybeLoadMore(s *Session) tea.Cmd {
	if sc.cursor >= len(sc.results)-1 && sc.nextCursor != "" && !sc.loadingMore {
		sc.loadingMore = true
		return sc.runSearch(s, sc.gen, sc.nextCursor, true)
	}
	return nil
}

func (sc *search) current() *steam.WorkshopItem {
	if sc.cursor < 0 || sc.cursor >= len(sc.results) {
		return nil
	}
	return &sc.results[sc.cursor]
}

func (sc *search) View(s *Session) string {
	th := s.Theme
	var b strings.Builder
	b.WriteString(sc.input.View() + "\n")

	status := th.Muted.Render(fmt.Sprintf("showing %d of %s", len(sc.results), humanize.Comma(int64(sc.total))))
	if sc.loading {
		status = sc.spinner.View() + th.Muted.Render(" searching…")
	}
	b.WriteString(status + "\n\n")

	installed := installedSet(s)
	h := max(3, s.BodyHeight()-7)
	start, end := listWindow(sc.cursor, len(sc.results), h)
	if start > 0 {
		b.WriteString(th.Muted.Render(fmt.Sprintf("  ↑ %d more", start)) + "\n")
	}
	for i := start; i < end; i++ {
		it := sc.results[i]
		sel := i == sc.cursor
		badge := "  "
		if installed[it.PublishedFileID] {
			badge = th.OK.Render("✓ ")
		}
		b.WriteString(renderRow(th, s.ContentWidth(), badge, it.Title, humanize.Bytes(uint64(it.FileSize)), sel) + "\n")
	}
	if end < len(sc.results) {
		b.WriteString(th.Muted.Render(fmt.Sprintf("  ↓ %d more", len(sc.results)-end)) + "\n")
	}
	if sc.loadingMore {
		b.WriteString(th.Muted.Render("loading more…") + "\n")
	}
	if len(sc.results) == 0 && !sc.loading {
		b.WriteString(th.Muted.Render("no results"))
	}

	hint := "enter: details   ↑/↓: browse   esc: back"
	if id := parseWorkshopID(sc.input.Value()); id != "" {
		hint += "   · ctrl+o: add ID " + id
	}
	b.WriteString("\n" + th.Muted.Render(hint))
	return pad(b.String())
}

func debounce(gen int) tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return debounceMsg{gen: gen} })
}

func installedSet(s *Session) map[string]bool {
	set := map[string]bool{}
	if s.Cfg != nil {
		for _, id := range s.Cfg.WorkshopItems() {
			set[id] = true
		}
	}
	return set
}
