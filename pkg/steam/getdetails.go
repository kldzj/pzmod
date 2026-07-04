package steam

import (
	"context"
	"strconv"
)

type detailsResponse struct {
	Response struct {
		PublishedFileDetails []WorkshopItem `json:"publishedfiledetails"`
	} `json:"response"`
}

// GetDetails fetches Workshop items by ID, serving cached entries where
// possible and batching the rest. Items the API reports as unavailable
// (result != 1) are returned in missing rather than items.
func (c *Client) GetDetails(ctx context.Context, ids []string) (items []WorkshopItem, missing []string, err error) {
	var toFetch []string
	for _, id := range ids {
		if id == "" {
			continue
		}
		if item, ok := c.cache.Get(id); ok {
			items = append(items, item)
			continue
		}
		toFetch = append(toFetch, id)
	}

	for _, batch := range chunk(toFetch, c.chunkSize) {
		fetched, miss, err := c.getDetailsChunk(ctx, batch)
		if err != nil {
			return nil, nil, err
		}
		missing = append(missing, miss...)
		for _, item := range fetched {
			c.cache.Set(item.PublishedFileID, item)
			items = append(items, item)
		}
	}

	return items, missing, nil
}

func (c *Client) getDetailsChunk(ctx context.Context, ids []string) (items []WorkshopItem, missing []string, err error) {
	q := c.baseQuery()
	q.Set("includechildren", "true")
	for i, id := range ids {
		q.Set("publishedfileids["+strconv.Itoa(i)+"]", id)
	}

	var resp detailsResponse
	if err := c.doGet(ctx, "/IPublishedFileService/GetDetails/v1/", q, &resp); err != nil {
		return nil, nil, err
	}

	for _, item := range resp.Response.PublishedFileDetails {
		if item.Result != 1 {
			missing = append(missing, item.PublishedFileID)
			continue
		}
		items = append(items, item)
	}
	return items, missing, nil
}

func chunk(s []string, size int) [][]string {
	if size <= 0 {
		size = 10
	}
	var out [][]string
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}
