package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/internal/build"
	"github.com/kldzj/pzmod/internal/domain"
)

// quickAdd fetches a Workshop item and adds it (plus its declared mods/maps) to
// the open config in memory. Collections route to the dependency resolver
// instead of being installed directly. The result is delivered as a toast (or a
// screen transition for collections/multi-mod items), so it works from any screen.
//
// When replaceTop is true, screen navigations use ReplaceMsg (atomic replace of
// the caller) rather than PushMsg, eliminating the Pop()+PushMsg race where a
// fast fetch causes the sheet to be pushed and immediately popped back off.
func quickAdd(s *Session, id string, replaceTop bool) tea.Cmd {
	show := func(scr Screen) tea.Msg {
		if replaceTop {
			return ReplaceMsg{Screen: scr}
		}
		return PushMsg{Screen: scr}
	}
	return s.Do(func(ctx context.Context) tea.Msg {
		items, _, err := s.Svc.Details(ctx, []string{id})
		if err != nil {
			return ErrMsg{Err: err}
		}
		if len(items) == 0 {
			return ErrMsg{Err: fmt.Errorf("item %s not found", id)}
		}
		it := items[0]
		if it.IsCollection() {
			return show(NewDeps([]string{id}))
		}
		parsed := it.Parse()
		if len(parsed.Maps) > 0 || len(parsed.Mods) > 1 {
			return show(NewAddSheet(it))
		}
		// Single mod, no maps: add directly.
		sm := s.Cfg.ServerMods().AddItem(it.PublishedFileID)
		explicit := s.Build() == build.B42
		for _, m := range parsed.Mods {
			sm = sm.AddMod(domain.FormatModRef(it.PublishedFileID, m, explicit))
		}
		s.Cfg.ApplyServerMods(sm)
		return modsChangedMsg{toast: "added " + itemTitle(&it) + " (unsaved)"}
	})
}
