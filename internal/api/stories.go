package api

import (
	"context"
	"fmt"
)

var storyEndpoints = map[StoryType]string{
	StoryTypeTop:  baseURL + "/topstories.json",
	StoryTypeNew:  baseURL + "/newstories.json",
	StoryTypeBest: baseURL + "/beststories.json",
	StoryTypeAsk:  baseURL + "/askstories.json",
	StoryTypeShow: baseURL + "/showstories.json",
	StoryTypeJobs: baseURL + "/jobstories.json",
}

// GetStoryIDs fetches the list of story IDs for a given story type.
func (c *Client) GetStoryIDs(ctx context.Context, st StoryType) ([]int, error) {
	url, ok := storyEndpoints[st]
	if !ok {
		return nil, fmt.Errorf("unknown story type: %s", st)
	}
	var ids []int
	if err := c.get(ctx, url, &ids); err != nil {
		return nil, fmt.Errorf("fetching %s stories: %w", st, err)
	}
	return ids, nil
}
