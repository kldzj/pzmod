// Package steamtest provides an in-memory fake of steam.API for service tests.
package steamtest

import (
	"context"
	"strings"

	"github.com/kldzj/pzmod/internal/steam"
)

// Fake is an in-memory steam.API. Populate Items keyed by published file ID.
// Unknown IDs are reported as missing by GetDetails.
type Fake struct {
	Items map[string]steam.WorkshopItem

	// DetailsErr/QueryErr, when set, are returned by the corresponding method.
	DetailsErr error
	QueryErr   error

	// DetailsCalls counts GetDetails invocations (to assert caching upstream).
	DetailsCalls int
	QueryCalls   int
}

// New returns a Fake seeded with the given items.
func New(items ...steam.WorkshopItem) *Fake {
	f := &Fake{Items: make(map[string]steam.WorkshopItem, len(items))}
	for _, it := range items {
		f.Items[it.PublishedFileID] = it
	}
	return f
}

var _ steam.API = (*Fake)(nil)

// GetDetails returns known items and reports unknown IDs as missing.
func (f *Fake) GetDetails(_ context.Context, ids []string) ([]steam.WorkshopItem, []string, error) {
	f.DetailsCalls++
	if f.DetailsErr != nil {
		return nil, nil, f.DetailsErr
	}
	var items []steam.WorkshopItem
	var missing []string
	for _, id := range ids {
		if id == "" {
			continue
		}
		if it, ok := f.Items[id]; ok {
			items = append(items, it)
		} else {
			missing = append(missing, id)
		}
	}
	return items, missing, nil
}

// QueryFiles returns items whose title contains the (case-insensitive) search
// text; with no search text it returns all items. Paging is ignored.
func (f *Fake) QueryFiles(_ context.Context, q steam.Query) (steam.Page, error) {
	f.QueryCalls++
	if f.QueryErr != nil {
		return steam.Page{}, f.QueryErr
	}
	var items []steam.WorkshopItem
	for _, it := range f.Items {
		if q.SearchText == "" || strings.Contains(strings.ToLower(it.Title), strings.ToLower(q.SearchText)) {
			items = append(items, it)
		}
	}
	return steam.Page{Items: items, Total: len(items)}, nil
}
