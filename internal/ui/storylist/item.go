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
	// Used by the list's built-in filter.
	return s.CommentStr() + " " + s.Domain()
}

func (s StoryItem) FilterValue() string {
	return s.Item.Title + " " + s.Item.By + " " + s.Domain()
}

// Domain returns the hostname from the story URL.
func (s StoryItem) Domain() string {
	if s.Item.URL == "" {
		return ""
	}
	u, err := url.Parse(s.Item.URL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// TimeAgo returns the relative timestamp.
func (s StoryItem) TimeAgo() string {
	return render.TimeAgo(s.Item.Time)
}

// CommentStr returns a formatted comment count string like HN.
func (s StoryItem) CommentStr() string {
	n := s.Item.Descendants
	switch {
	case n == 0:
		return "discuss"
	case n == 1:
		return "1 comment"
	default:
		return fmt.Sprintf("%d comments", n)
	}
}
