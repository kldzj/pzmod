package steam

import (
	"context"
	"strconv"
)

// Query types (EPublishedFileQueryType). Verified against the live API for
// appid 108600: 12 returns relevance-ranked results for a text search, 3 ranks
// by trend for browsing without a query.
const (
	queryRankedByTrend      = 3
	queryRankedByTextSearch = 12
)

// Query parameterizes a Workshop search.
type Query struct {
	// SearchText is the free-text query. When empty, results are browsed by
	// trend instead of ranked by relevance.
	SearchText string
	// Tags restricts results to items carrying all of these Workshop tags
	// (e.g. "Mod", "Build 41").
	Tags []string
	// PerPage is the page size (default 20, max 100 per the API).
	PerPage int
	// Cursor pages through results; use "" or "*" for the first page and the
	// returned NextCursor for subsequent pages.
	Cursor string
}

// Page is a single page of search results.
type Page struct {
	Items      []WorkshopItem
	Total      int
	NextCursor string
}

type queryResponse struct {
	Response struct {
		Total                int            `json:"total"`
		NextCursor           string         `json:"next_cursor"`
		PublishedFileDetails []WorkshopItem `json:"publishedfiledetails"`
	} `json:"response"`
}

// QueryFiles searches the Workshop for the configured app, returning one page.
func (c *Client) QueryFiles(ctx context.Context, q Query) (Page, error) {
	perPage := q.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	cursor := q.Cursor
	if cursor == "" {
		cursor = "*"
	}

	queryType := queryRankedByTrend
	if q.SearchText != "" {
		queryType = queryRankedByTextSearch
	}

	v := c.baseQuery()
	v.Set("appid", strconv.Itoa(c.appID))
	v.Set("query_type", strconv.Itoa(queryType))
	v.Set("cursor", cursor)
	v.Set("numperpage", strconv.Itoa(perPage))
	v.Set("return_metadata", "true")
	v.Set("return_short_description", "true")
	v.Set("return_children", "true")
	if q.SearchText != "" {
		v.Set("search_text", q.SearchText)
	}
	for i, tag := range q.Tags {
		v.Set("requiredtags["+strconv.Itoa(i)+"]", tag)
	}

	var resp queryResponse
	if err := c.doGet(ctx, "/IPublishedFileService/QueryFiles/v1/", v, &resp); err != nil {
		return Page{}, err
	}

	return Page{
		Items:      resp.Response.PublishedFileDetails,
		Total:      resp.Response.Total,
		NextCursor: resp.Response.NextCursor,
	}, nil
}
