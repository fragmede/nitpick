package api

import (
	"encoding/json"
)

// StoryType represents the different HN story categories.
type StoryType string

const (
	StoryTypeTop      StoryType = "top"
	StoryTypeNew      StoryType = "new"
	StoryTypeBest     StoryType = "best"
	StoryTypeAsk      StoryType = "ask"
	StoryTypeShow     StoryType = "show"
	StoryTypeJobs     StoryType = "jobs"
	StoryTypeThreads  StoryType = "threads"
	StoryTypePast     StoryType = "past"
	StoryTypeComments StoryType = "comments"
)

// Item represents an HN item (story, comment, job, poll, pollopt).
type Item struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	By          string `json:"by"`
	Time        int64  `json:"time"`
	Text        string `json:"text"`
	Parent      int    `json:"parent"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Score       int    `json:"score"`
	Descendants int    `json:"descendants"`
	Dead        bool   `json:"dead"`
	Deleted     bool   `json:"deleted"`
	Poll        int    `json:"poll"`

	// Kids is stored as a JSON array of ints.
	// We use json.RawMessage to handle the raw JSON and parse lazily.
	RawKids json.RawMessage `json:"kids"`
	RawParts json.RawMessage `json:"parts"`

	// Parsed kids/parts (populated after decode).
	kids  []int
	parts []int
}

// Kids returns the child item IDs for this item.
func (it *Item) Kids() []int {
	if it.kids != nil {
		return it.kids
	}
	if len(it.RawKids) == 0 {
		return nil
	}
	_ = json.Unmarshal(it.RawKids, &it.kids)
	return it.kids
}

// Parts returns the poll option IDs for this item.
func (it *Item) Parts() []int {
	if it.parts != nil {
		return it.parts
	}
	if len(it.RawParts) == 0 {
		return nil
	}
	_ = json.Unmarshal(it.RawParts, &it.parts)
	return it.parts
}

// KidsJSON returns the raw JSON for kids (for cache storage).
func (it *Item) KidsJSON() string {
	if len(it.RawKids) == 0 {
		return "[]"
	}
	return string(it.RawKids)
}

// User represents an HN user profile.
type User struct {
	ID        string `json:"id"`
	Created   int64  `json:"created"`
	Karma     int    `json:"karma"`
	About     string `json:"about"`
	Submitted []int  `json:"submitted"`
}
