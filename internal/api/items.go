package api

import "context"

// GetStories fetches story IDs for the given type and batch-fetches the items.
// limit controls how many items to fetch (0 = all).
func (c *Client) GetStories(ctx context.Context, st StoryType, limit int) ([]*Item, error) {
	ids, err := c.GetStoryIDs(ctx, st)
	if err != nil {
		return nil, err
	}
	if limit > 0 && limit < len(ids) {
		ids = ids[:limit]
	}
	return c.BatchGetItems(ctx, ids)
}
