package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	baseURL        = "https://hacker-news.firebaseio.com/v0"
	requestTimeout = 10 * time.Second
	maxConcurrent  = 10
)

// Client is the HN API client.
type Client struct {
	http *http.Client
}

// NewClient creates a new HN API client.
func NewClient() *Client {
	return &Client{
		http: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// get fetches a URL and decodes the JSON response into dst.
func (c *Client) get(ctx context.Context, url string, dst interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "nitpick/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, url, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decoding response from %s: %w", url, err)
	}
	return nil
}

// GetItem fetches a single item by ID.
func (c *Client) GetItem(ctx context.Context, id int) (*Item, error) {
	url := fmt.Sprintf("%s/item/%d.json", baseURL, id)
	var item Item
	if err := c.get(ctx, url, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// BatchGetItems fetches multiple items concurrently with a concurrency limit.
// Returns items in the same order as the input IDs. Failed fetches are nil.
func (c *Client) BatchGetItems(ctx context.Context, ids []int) ([]*Item, error) {
	results := make([]*Item, len(ids))
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	for i, id := range ids {
		i, id := i, id
		g.Go(func() error {
			item, err := c.GetItem(ctx, id)
			if err != nil {
				// Non-fatal: individual items can fail.
				return nil
			}
			mu.Lock()
			results[i] = item
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// GetUser fetches a user profile by username.
func (c *Client) GetUser(ctx context.Context, username string) (*User, error) {
	url := fmt.Sprintf("%s/user/%s.json", baseURL, username)
	var user User
	if err := c.get(ctx, url, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetMaxItem returns the current largest item ID.
func (c *Client) GetMaxItem(ctx context.Context) (int, error) {
	url := baseURL + "/maxitem.json"
	var maxID int
	if err := c.get(ctx, url, &maxID); err != nil {
		return 0, err
	}
	return maxID, nil
}
