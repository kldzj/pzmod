package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/internal/build"
	"github.com/kldzj/pzmod/internal/domain"
	"github.com/kldzj/pzmod/internal/openurl"
	"github.com/kldzj/pzmod/internal/steam"
)

// TUI-local finding codes - not produced by the service layer.
const codeMapOrder = "tui.maporder"
const codeModOrder = "tui.modorder"

// validate runs the validation pipeline against the (possibly unsaved) config -
// a live dry-run - and shows categorized, actionable findings.
type validate struct {
	loading  bool
	load     loader
	report   domain.Report
	findings []domain.Finding
	cursor   int
	filter   filterState
}

// NewValidate returns the validation screen.
func NewValidate() Screen { return &validate{loading: true, load: newLoader()} }

func (v *validate) Title() string { return "Validate" }

type validateMsg struct {
	report domain.Report
	extra  []domain.Finding
	err    error
}

type revalidateMsg struct{}

func (v *validate) Init(s *Session) tea.Cmd { return tea.Batch(v.load.tick(), v.run(s)) }

func (v *validate) run(s *Session) tea.Cmd {
	v.loading = true
	sm := s.Cfg.ServerMods()
	b := s.Build()
	profile := s.Profile // captured pointer; may be nil if no profile open
	return s.Do(func(ctx context.Context) tea.Msg {
		report, err := s.Svc.Validate(ctx, sm, b)
		if err != nil {
			return validateMsg{err: err}
		}
		var extra []domain.Finding
		if mp := domain.SuggestMapOrder(sm.Maps); len(mp.Moved) > 0 {
			extra = append(extra, domain.Finding{
				Code:     codeMapOrder,
				Severity: domain.SeverityWarning,
				Message:  fmt.Sprintf("Map order: %d map(s) should move so the base map loads last", len(mp.Moved)),
			})
		}
		if profile != nil {
			if lp, lerr := s.Svc.SuggestLoadOrder(ctx, sm, *profile); lerr == nil && len(lp.Moved) > 0 {
				extra = append(extra, domain.Finding{
					Code:     codeModOrder,
					Severity: domain.SeverityInfo,
					Message:  fmt.Sprintf("Load order: %d mod(s) could be reordered (frameworks/dependencies first)", len(lp.Moved)),
				})
			}
		}
		return validateMsg{report: report, extra: extra}
	})
}

func (v *validate) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	if cmd, ok := v.load.update(msg); ok {
		if v.loading {
			return v, cmd
		}
		return v, nil
	}
	switch msg := msg.(type) {
	case validateMsg:
		if msg.err != nil {
			return v, tea.Batch(Fail(msg.err), Pop())
		}
		v.loading = false
		v.report = msg.report
		v.findings = append(msg.report.Sorted(), msg.extra...)
		r := msg.report
		s.LastValidation = &r // cache for the dashboard status line
		v.clampCursor()
		return v, nil
	case revalidateMsg:
		return v, tea.Batch(v.load.tick(), v.run(s))
	case resumedMsg:
		// Re-run when returning from a screen a fix navigated to (Load order via
		// the mod-order advisory, or the deps resolver via 'r'), so the findings
		// reflect changes made there instead of going stale.
		return v, tea.Batch(v.load.tick(), v.run(s))
	case tea.KeyMsg:
		if v.filter.active {
			if v.filter.handleKey(msg) {
				v.clampCursor()
				return v, nil
			}
		}
		switch msg.String() {
		case "esc":
			if v.filter.has() {
				v.filter.clear()
				v.clampCursor()
				return v, nil
			}
			return v, Pop()
		case "/":
			v.filter.start()
			return v, nil
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
			}
		case "down", "j":
			if v.cursor < len(v.shownFindings())-1 {
				v.cursor++
			}
		case "pgup":
			h := max(3, s.BodyHeight()-7)
			v.cursor = max(0, v.cursor-h)
		case "pgdown":
			h := max(3, s.BodyHeight()-7)
			if n := len(v.shownFindings()); n > 0 {
				v.cursor = min(n-1, v.cursor+h)
			}
		case "home":
			v.cursor = 0
		case "end":
			if n := len(v.shownFindings()); n > 0 {
				v.cursor = n - 1
			}
		case "o":
			if f, ok := v.current(); ok {
				if id := findingItemID(f); id != "" {
					item := steam.WorkshopItem{PublishedFileID: id}
					_ = openurl.Open(item.WorkshopURL())
					return v, Toast("opening in browser…")
				}
			}
			return v, Toast("no Workshop page for this finding")
		case "r":
			return v, Push(NewDeps(s.Cfg.WorkshopItems())) // resolve all missing
		case "enter":
			if f, ok := v.current(); ok {
				return v, v.fix(s, f)
			}
		}
	}
	return v, nil
}

// shownFindings returns the findings matching the current filter (message +
// subject; Severity is an int, not text, so it is not matched).
func (v *validate) shownFindings() []domain.Finding {
	if !v.filter.has() {
		return v.findings
	}
	var out []domain.Finding
	for _, f := range v.findings {
		if filterMatch(v.filter.query, f.Message, f.Subject) {
			out = append(out, f)
		}
	}
	return out
}

func (v *validate) clampCursor() {
	if n := len(v.shownFindings()); v.cursor >= n {
		v.cursor = max(0, n-1)
	}
	if v.cursor < 0 {
		v.cursor = 0
	}
}

func (v *validate) current() (domain.Finding, bool) {
	sh := v.shownFindings()
	if v.cursor < 0 || v.cursor >= len(sh) {
		return domain.Finding{}, false
	}
	return sh[v.cursor], true
}

// fix applies a targeted remedy for the selected finding, then re-validates.
func (v *validate) fix(s *Session, f domain.Finding) tea.Cmd {
	switch f.Code {
	case domain.CodeMissingDependency:
		id := f.Subject
		return s.Do(func(ctx context.Context) tea.Msg {
			plan, err := s.Svc.Resolve(ctx, []string{id}, s.Cfg.ServerMods())
			if err != nil {
				return ErrMsg{Err: err}
			}
			s.Cfg.ApplyServerMods(plan.Apply(s.Cfg.ServerMods(), s.Build() == build.B42))
			return revalidateMsg{}
		})
	case domain.CodeUnusedModID:
		s.Cfg.ApplyServerMods(s.Cfg.ServerMods().AddMod(domain.FormatModRef("", f.Subject, s.Build() == build.B42)))
		return tea.Batch(Toast("enabled "+f.Subject), func() tea.Msg { return revalidateMsg{} })
	case domain.CodeUnusedMap:
		s.Cfg.ApplyServerMods(s.Cfg.ServerMods().AddMap(f.Subject))
		return tea.Batch(Toast("enabled map "+f.Subject), func() tea.Msg { return revalidateMsg{} })
	case domain.CodeUnknownModID:
		s.Cfg.ApplyServerMods(s.Cfg.ServerMods().RemoveMod(f.Subject))
		return tea.Batch(Toast("removed "+f.Subject), func() tea.Msg { return revalidateMsg{} })
	case domain.CodeDelisted, domain.CodeBanned:
		id := f.Subject
		return Confirm("Remove workshop item "+id+"?", func() tea.Msg {
			s.Cfg.ApplyServerMods(s.Cfg.ServerMods().RemoveItem(id))
			return revalidateMsg{}
		})
	case codeMapOrder:
		plan := domain.SuggestMapOrder(s.Cfg.Maps())
		s.Cfg.SetMaps(plan.Ordered)
		return tea.Batch(Toast("map order updated"), func() tea.Msg { return revalidateMsg{} })
	case codeModOrder:
		return Push(NewLoadOrder())
	default:
		if findingItemID(f) != "" {
			return Toast("no automatic fix - press o to open the mod's page")
		}
		return Toast("no automatic fix for this finding")
	}
}

// findingItemID returns the finding's Workshop item ID if its Subject is one
// (numeric), else "". Used to offer "open in Steam" for item-level findings.
func findingItemID(f domain.Finding) string { return parseWorkshopID(f.Subject) }

func actionable(code string) bool {
	switch code {
	case domain.CodeMissingDependency, domain.CodeUnusedModID, domain.CodeUnusedMap,
		domain.CodeUnknownModID, domain.CodeDelisted, domain.CodeBanned,
		codeMapOrder, codeModOrder:
		return true
	}
	return false
}

// countSeverity counts findings at the given severity level.
func countSeverity(findings []domain.Finding, sev domain.Severity) int {
	n := 0
	for _, f := range findings {
		if f.Severity == sev {
			n++
		}
	}
	return n
}

func (v *validate) View(s *Session) string {
	th := s.Theme
	if v.loading {
		return pad(v.load.view(th, "validating…"))
	}
	var b strings.Builder
	if s.Build() == build.B42 {
		b.WriteString(th.Badge.Render("BUILD 42") + " " +
			th.Muted.Render("multiplayer still disables mods as of now") + "\n\n")
	}

	if len(v.findings) == 0 {
		b.WriteString(th.OK.Render("✓ no problems found") + "\n\n")
		b.WriteString(th.Muted.Render("esc: back"))
		return pad(b.String())
	}

	if line := v.filter.view(th); line != "" {
		b.WriteString(line + "\n\n")
	}
	shown := v.shownFindings()
	if len(shown) == 0 {
		b.WriteString(th.Muted.Render(fmt.Sprintf("no matches for %q", v.filter.query)) + "\n\n")
	}
	h := max(3, s.BodyHeight()-7-v.filter.chrome())
	start, end := listWindow(v.cursor, len(shown), h)
	if start > 0 {
		b.WriteString(th.Muted.Render(fmt.Sprintf("  ↑ %d more", start)) + "\n")
	}
	for i := start; i < end; i++ {
		f := shown[i]
		sel := i == v.cursor
		prefix := cursorPrefix(th, sel) + severityTagFor(th, f.Severity) + " "
		right := "manual"
		if f.Code == codeModOrder {
			right = "↵ open"
		} else if actionable(f.Code) {
			right = "↵ fix"
		}
		b.WriteString(renderRow(th, s.ContentWidth(), prefix, f.Message, right, sel) + "\n")
	}
	if end < len(shown) {
		b.WriteString(th.Muted.Render(fmt.Sprintf("  ↓ %d more", len(shown)-end)) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s  %s  %s\n",
		th.Error.Render(fmt.Sprintf("%d errors", countSeverity(shown, domain.SeverityError))),
		th.Warn.Render(fmt.Sprintf("%d warnings", countSeverity(shown, domain.SeverityWarning))),
		th.Muted.Render(fmt.Sprintf("%d info", countSeverity(shown, domain.SeverityInfo)))))
	b.WriteString(th.Muted.Render("↵: fix selected   o: open page   r: resolve all missing   /: filter   esc: back"))
	return pad(b.String())
}

func severityTagFor(th Theme, sev domain.Severity) string {
	switch sev {
	case domain.SeverityError:
		return th.Error.Render("ERROR")
	case domain.SeverityWarning:
		return th.Warn.Render("WARN ")
	default:
		return th.Subtitle.Render("INFO ")
	}
}
