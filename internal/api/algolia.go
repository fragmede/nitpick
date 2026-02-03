package api

import (
	"context"
	"fmt"
	"time"
)

const algoliaBaseURL = "https://hn.algolia.com/api/v1"

// AlgoliaResponse is the search response from the Algolia HN API.
type AlgoliaResponse struct {
	Hits []AlgoliaHit `json:"hits"`
}

// AlgoliaHit is a single search result.
type AlgoliaHit struct {
	ObjectID    string `json:"objectID"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Author      string `json:"author"`
	Points      int    `json:"points"`
	NumComments int    `json:"num_comments"`
	CreatedAtI  int64  `json:"created_at_i"`
	StoryText   string `json:"story_text"`
	CommentText string `json:"comment_text"`
	ParentID    int    `json:"parent_id"`
	StoryID     int    `json:"story_id"`
	StoryTitle  string `json:"story_title"`
	StoryURL    string `json:"story_url"`
}

// ToItem converts an Algolia hit to an api.Item.
func (h AlgoliaHit) ToItem() *Item {
	var id int
	fmt.Sscanf(h.ObjectID, "%d", &id)

	item := &Item{
		ID:    id,
		By:    h.Author,
		Time:  h.CreatedAtI,
		Score: h.Points,
	}

	if h.Title != "" {
		// It's a story.
		item.Type = "story"
		item.Title = h.Title
		item.URL = h.URL
		item.Descendants = h.NumComments
		item.Text = h.StoryText
	} else {
		// It's a comment.
		item.Type = "comment"
		item.Text = h.CommentText
		item.Parent = h.ParentID
	}

	return item
}

// GetPastStories fetches yesterday's front page stories via Algolia.
func (c *Client) GetPastStories(ctx context.Context, limit int) ([]*Item, error) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	startOfYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	endOfYesterday := startOfYesterday.AddDate(0, 0, 1)

	url := fmt.Sprintf("%s/search?tags=front_page&numericFilters=created_at_i>%d,created_at_i<%d&hitsPerPage=%d",
		algoliaBaseURL, startOfYesterday.Unix(), endOfYesterday.Unix(), limit)

	var resp AlgoliaResponse
	if err := c.get(ctx, url, &resp); err != nil {
		return nil, fmt.Errorf("fetching past stories: %w", err)
	}

	items := make([]*Item, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		items = append(items, hit.ToItem())
	}
	return items, nil
}

// GetNewestComments fetches the newest comments site-wide via Algolia.
func (c *Client) GetNewestComments(ctx context.Context, limit int) ([]*Item, error) {
	url := fmt.Sprintf("%s/search_by_date?tags=comment&hitsPerPage=%d",
		algoliaBaseURL, limit)

	var resp AlgoliaResponse
	if err := c.get(ctx, url, &resp); err != nil {
		return nil, fmt.Errorf("fetching newest comments: %w", err)
	}

	items := make([]*Item, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		item := hit.ToItem()
		item.StoryTitle = hit.StoryTitle
		if hit.StoryID != 0 {
			item.Parent = hit.StoryID
		}
		items = append(items, item)
	}
	return items, nil
}

// GetUserThreads fetches the user's recent comments, matching HN's
// /threads?id=username page. Each comment includes the parent story title.
func (c *Client) GetUserThreads(ctx context.Context, username string, limit int) ([]*Item, error) {
	url := fmt.Sprintf("%s/search_by_date?tags=comment,author_%s&hitsPerPage=%d",
		algoliaBaseURL, username, limit)

	var resp AlgoliaResponse
	if err := c.get(ctx, url, &resp); err != nil {
		return nil, fmt.Errorf("fetching user threads: %w", err)
	}

	items := make([]*Item, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		item := hit.ToItem()
		item.StoryTitle = hit.StoryTitle
		items = append(items, item)
	}
	return items, nil
}
