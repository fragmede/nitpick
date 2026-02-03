package storylist

import (
	"fmt"
	"net/url"

	"github.com/fragmede/hn-tui/internal/api"
	"github.com/fragmede/hn-tui/internal/render"
)

// StoryItem wraps an API item for the bubbles list.
type StoryItem struct {
	*api.Item
	Index int
}

func (s StoryItem) Title() string {
	if s.Item.Title != "" {
		return s.Item.Title
	}
	return fmt.Sprintf("[%s]", s.Item.Type)
}

func (s StoryItem) Description() string {
	parts := make([]string, 0, 4)

	if s.Item.Score > 0 {
		parts = append(parts, fmt.Sprintf("%d points", s.Item.Score))
	}
	if s.Item.By != "" {
		parts = append(parts, fmt.Sprintf("by %s", s.Item.By))
	}
	parts = append(parts, render.TimeAgo(s.Item.Time))
	if s.Item.Descendants > 0 {
		parts = append(parts, fmt.Sprintf("%d comments", s.Item.Descendants))
	}

	desc := ""
	for i, p := range parts {
		if i > 0 {
			desc += " | "
		}
		desc += p
	}

	if s.Item.URL != "" {
		if u, err := url.Parse(s.Item.URL); err == nil {
			desc += "  (" + u.Host + ")"
		}
	}
	return desc
}

func (s StoryItem) FilterValue() string {
	return s.Item.Title + " " + s.Item.By
}
