package steam

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLive exercises the real Steam Web API. It is skipped unless PZMOD_LIVE=1
// and a key is provided via PZMOD_STEAM_KEY, so normal `go test` stays offline.
func TestLive(t *testing.T) {
	if os.Getenv("PZMOD_LIVE") != "1" {
		t.Skip("set PZMOD_LIVE=1 and PZMOD_STEAM_KEY to run live API tests")
	}
	key := os.Getenv("PZMOD_STEAM_KEY")
	if key == "" {
		t.Skip("PZMOD_STEAM_KEY not set")
	}

	client := New(key)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// A known mod with a Mod ID in its description.
	items, missing, err := client.GetDetails(ctx, []string{"2849247394"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, %v missing", len(items), missing)
	}
	it := items[0]
	t.Logf("title=%q filetype=%d size=%d updated=%d", it.Title, it.FileType, it.FileSize, it.TimeUpdated)
	parsed := it.Parse()
	t.Logf("parsed mods=%v maps=%v children=%d", parsed.Mods, parsed.Maps, len(it.Children))

	// Search.
	page, err := client.QueryFiles(ctx, Query{SearchText: "hydrocraft", PerPage: 3})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("search total=%d next=%q got=%d", page.Total, page.NextCursor, len(page.Items))
	if page.Total == 0 {
		t.Errorf("expected search results for 'hydrocraft'")
	}
	for _, p := range page.Items {
		t.Logf("  %s | %s | preview=%s", p.PublishedFileID, p.Title, p.PreviewURL)
	}
}
