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
		// Stash story context so we can display it.
		if hit.StoryTitle != "" {
			item.Title = hit.StoryTitle
		}
		if hit.StoryID != 0 {
			item.Parent = hit.StoryID
		}
		items = append(items, item)
	}
	return items, nil
}

// GetUserThreads fetches stories the user has commented on.
// Fetches user's recent submissions, filters for comments, returns parent stories.
func (c *Client) GetUserThreads(ctx context.Context, username string, limit int) ([]*Item, error) {
	user, err := c.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	// Take recent submitted items.
	submitted := user.Submitted
	fetchCount := limit * 3 // Fetch more since many will be non-comments.
	if fetchCount > len(submitted) {
		fetchCount = len(submitted)
	}

	items, err := c.BatchGetItems(ctx, submitted[:fetchCount])
	if err != nil {
		return nil, fmt.Errorf("fetching user items: %w", err)
	}

	// Collect unique parent story IDs from comments.
	storyIDs := make(map[int]bool)
	var orderedStoryIDs []int
	for _, item := range items {
		if item == nil || item.Type != "comment" {
			continue
		}
		sid := findRootStoryID(item, c, ctx)
		if sid != 0 && !storyIDs[sid] {
			storyIDs[sid] = true
			orderedStoryIDs = append(orderedStoryIDs, sid)
		}
		if len(orderedStoryIDs) >= limit {
			break
		}
	}

	// Fetch the actual stories.
	stories, err := c.BatchGetItems(ctx, orderedStoryIDs)
	if err != nil {
		return nil, fmt.Errorf("fetching thread stories: %w", err)
	}
	return stories, nil
}

// findRootStoryID walks up the parent chain to find the root story.
func findRootStoryID(item *Item, c *Client, ctx context.Context) int {
	current := item
	seen := make(map[int]bool)
	for current.Parent != 0 {
		if seen[current.Parent] {
			return current.Parent
		}
		seen[current.Parent] = true
		parent, err := c.GetItem(ctx, current.Parent)
		if err != nil || parent == nil {
			return current.Parent
		}
		if parent.Type == "story" {
			return parent.ID
		}
		current = parent
	}
	return current.ID
}
