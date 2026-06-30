package steam

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testClient(t *testing.T, h http.HandlerFunc, opts ...Option) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	all := append([]Option{WithBaseURL(srv.URL), WithRateLimiter(nil)}, opts...)
	return New("TESTKEY", all...)
}

// requestedIDs extracts publishedfileids[N] params from a GetDetails request.
func requestedIDs(v url.Values) []string {
	var ids []string
	for i := 0; ; i++ {
		id := v.Get(fmt.Sprintf("publishedfileids[%d]", i))
		if id == "" {
			break
		}
		ids = append(ids, id)
	}
	return ids
}

func TestGetDetailsChunking(t *testing.T) {
	var requests int
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Query().Get("includechildren") != "true" {
			t.Errorf("includechildren not set")
		}
		ids := requestedIDs(r.URL.Query())
		var sb strings.Builder
		sb.WriteString(`{"response":{"publishedfiledetails":[`)
		for i, id := range ids {
			if i > 0 {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, `{"result":1,"publishedfileid":%q,"file_size":123}`, id)
		}
		sb.WriteString(`]}}`)
		w.Write([]byte(sb.String()))
	})

	var ids []string
	for i := 0; i < 25; i++ {
		ids = append(ids, fmt.Sprintf("%d", i))
	}
	items, missing, err := client.GetDetails(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 25 {
		t.Errorf("items = %d; want 25", len(items))
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v; want none", missing)
	}
	if requests != 3 { // 10 + 10 + 5
		t.Errorf("requests = %d; want 3 (chunked at 10)", requests)
	}
}

func TestItemSizeBothForms(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"response":{"publishedfiledetails":[
			{"result":1,"publishedfileid":"a","file_size":4096},
			{"result":1,"publishedfileid":"b","file_size":"8192"}
		]}}`))
	})
	items, _, err := client.GetDetails(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]ItemSize{}
	for _, it := range items {
		got[it.PublishedFileID] = it.FileSize
	}
	if got["a"] != 4096 || got["b"] != 8192 {
		t.Errorf("file sizes = %v; want a=4096 b=8192", got)
	}
}

func TestGetDetailsMissing(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"response":{"publishedfiledetails":[
			{"result":1,"publishedfileid":"ok"},
			{"result":9,"publishedfileid":"gone"}
		]}}`))
	})
	items, missing, err := client.GetDetails(context.Background(), []string{"ok", "gone"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].PublishedFileID != "ok" {
		t.Errorf("items = %v; want [ok]", items)
	}
	if len(missing) != 1 || missing[0] != "gone" {
		t.Errorf("missing = %v; want [gone]", missing)
	}
}

func TestUnauthorized(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, _, err := client.GetDetails(context.Background(), []string{"x"})
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Errorf("err = %v; want ErrInvalidAPIKey", err)
	}
}

func TestCacheTTL(t *testing.T) {
	var requests int
	cur := time.Unix(1000, 0)
	clock := func() time.Time { return cur }
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Write([]byte(`{"response":{"publishedfiledetails":[{"result":1,"publishedfileid":"x"}]}}`))
	}, WithClock(clock))

	if _, _, err := client.GetDetails(context.Background(), []string{"x"}); err != nil {
		t.Fatal(err)
	}
	// Second call within TTL is served from cache.
	if _, _, err := client.GetDetails(context.Background(), []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Errorf("requests = %d; want 1 (cached within TTL)", requests)
	}

	// Advance past the 5m TTL and refetch.
	cur = cur.Add(6 * time.Minute)
	if _, _, err := client.GetDetails(context.Background(), []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Errorf("requests = %d; want 2 (refetch after TTL)", requests)
	}
}

func TestQueryFilesTextSearch(t *testing.T) {
	var captured url.Values
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query()
		w.Write([]byte(`{"response":{"total":42,"next_cursor":"NEXT","publishedfiledetails":[
			{"result":1,"publishedfileid":"1","title":"Hydrocraft"},
			{"result":1,"publishedfileid":"2","title":"Hydro XS"}
		]}}`))
	})

	page, err := client.QueryFiles(context.Background(), Query{SearchText: "hydro", Tags: []string{"Mod"}, PerPage: 2})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Get("query_type") != "12" {
		t.Errorf("query_type = %q; want 12 (text search)", captured.Get("query_type"))
	}
	if captured.Get("search_text") != "hydro" {
		t.Errorf("search_text = %q", captured.Get("search_text"))
	}
	if captured.Get("requiredtags[0]") != "Mod" {
		t.Errorf("requiredtags[0] = %q; want Mod", captured.Get("requiredtags[0]"))
	}
	if captured.Get("cursor") != "*" {
		t.Errorf("cursor = %q; want *", captured.Get("cursor"))
	}
	if captured.Get("appid") != "108600" {
		t.Errorf("appid = %q; want 108600", captured.Get("appid"))
	}
	if page.Total != 42 || page.NextCursor != "NEXT" || len(page.Items) != 2 {
		t.Errorf("page = %+v", page)
	}
}

func TestQueryFilesBrowseDefault(t *testing.T) {
	var captured url.Values
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query()
		w.Write([]byte(`{"response":{"total":0,"publishedfiledetails":[]}}`))
	})
	if _, err := client.QueryFiles(context.Background(), Query{}); err != nil {
		t.Fatal(err)
	}
	if captured.Get("query_type") != "3" {
		t.Errorf("query_type = %q; want 3 (trend browse)", captured.Get("query_type"))
	}
	if captured.Has("search_text") {
		t.Errorf("search_text should be absent for browse")
	}
}
