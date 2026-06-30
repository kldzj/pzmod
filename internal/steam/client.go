// Package steam is an injectable client for the Steam Web API's Workshop
// endpoints. It replaces the v2 package-level API key and cache globals with a
// Client constructed via New, so it is safe to test and free of shared state.
package steam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

// AppID is the Steam application ID for Project Zomboid.
const AppID = 108600

const defaultBaseURL = "https://api.steampowered.com"

// ErrInvalidAPIKey is returned when the Steam API rejects the key (HTTP 401).
var ErrInvalidAPIKey = errors.New("steam api key is invalid")

// API is the consumer-facing interface the services depend on, so they can be
// tested against a fake (see steamtest).
type API interface {
	GetDetails(ctx context.Context, ids []string) (items []WorkshopItem, missing []string, err error)
	QueryFiles(ctx context.Context, q Query) (Page, error)
}

// Client talks to the Steam Web API. Construct it with New.
type Client struct {
	apiKey    string
	appID     int
	http      *http.Client
	cache     Cache
	chunkSize int
	baseURL   string
	limiter   *rate.Limiter
	now       func() time.Time
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// WithCache overrides the cache.
func WithCache(cache Cache) Option { return func(c *Client) { c.cache = cache } }

// WithBaseURL overrides the API base URL (used to point at a test server).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithClock overrides the clock used by the default cache.
func WithClock(now func() time.Time) Option { return func(c *Client) { c.now = now } }

// WithChunkSize overrides the GetDetails request batch size (default 10).
func WithChunkSize(n int) Option {
	return func(c *Client) {
		if n > 0 {
			c.chunkSize = n
		}
	}
}

// WithRateLimiter overrides the request rate limiter (nil disables limiting).
func WithRateLimiter(l *rate.Limiter) Option { return func(c *Client) { c.limiter = l } }

// New returns a Client for the given Steam Web API key.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:    apiKey,
		appID:     AppID,
		chunkSize: 10,
		baseURL:   defaultBaseURL,
		limiter:   rate.NewLimiter(rate.Every(200*time.Millisecond), 5),
		now:       time.Now,
	}
	for _, o := range opts {
		o(c)
	}
	if c.http == nil {
		rc := retryablehttp.NewClient()
		rc.Logger = nil
		rc.RetryMax = 3
		c.http = rc.StandardClient()
	}
	if c.cache == nil {
		c.cache = NewMemCache(5*time.Minute, c.now)
	}
	return c
}

// doGet performs a rate-limited GET against the API and decodes JSON into out.
func (c *Client) doGet(ctx context.Context, endpoint string, query url.Values, out any) error {
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}
	}

	u := c.baseURL + endpoint + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return ErrInvalidAPIKey
		}
		return fmt.Errorf("steam api request failed with status %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// baseQuery returns a query value set seeded with the API key.
func (c *Client) baseQuery() url.Values {
	v := url.Values{}
	v.Set("key", c.apiKey)
	return v
}
