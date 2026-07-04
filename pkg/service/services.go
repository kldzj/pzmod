// Package service orchestrates the domain logic over injected adapters (Steam
// API, on-disk store, optional mod.info provider). Services are stateless and
// return plain result/plan structs: they never print and never prompt, so the
// same calls power both the CLI and the TUI.
package service

import (
	"context"
	"time"

	"github.com/kldzj/pzmod/pkg/modinfo"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/store"
)

// Services is the dependency-injected aggregate the presentation layers use.
type Services struct {
	Steam steam.API
	Store *store.Store

	// ModInfoOverride, when set, is used instead of building a provider from a
	// profile's WorkshopContentPath (tests inject a fake here).
	ModInfoOverride modinfo.Provider

	// Now is the clock used for backup notes/timestamps.
	Now func() time.Time
}

// New constructs a Services aggregate.
func New(steamAPI steam.API, st *store.Store) *Services {
	return &Services{Steam: steamAPI, Store: st, Now: time.Now}
}

// providerFor returns the mod.info provider for a profile (or the test override).
func (s *Services) providerFor(p store.Profile) modinfo.Provider {
	if s.ModInfoOverride != nil {
		return s.ModInfoOverride
	}
	return modinfo.NewProvider(p.WorkshopContentPath)
}

// Search runs a Workshop search.
func (s *Services) Search(ctx context.Context, q steam.Query) (steam.Page, error) {
	return s.Steam.QueryFiles(ctx, q)
}

// Details fetches Workshop items by ID (for browse/detail views).
func (s *Services) Details(ctx context.Context, ids []string) ([]steam.WorkshopItem, []string, error) {
	return s.Steam.GetDetails(ctx, ids)
}

// SnapshotProfile backs up a profile's config file and prunes to its retention.
func (s *Services) SnapshotProfile(p store.Profile, note, kind string) (store.BackupEntry, error) {
	entry, err := s.Store.Snapshot(p.ID, p.IniPath, note, kind)
	if err != nil {
		return store.BackupEntry{}, err
	}
	keep := p.BackupRetention
	if keep <= 0 {
		keep = store.DefaultBackupRetention
	}
	if err := s.Store.Prune(p.ID, keep); err != nil {
		return entry, err
	}
	return entry, nil
}
