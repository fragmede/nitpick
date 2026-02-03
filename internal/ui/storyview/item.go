package storyview

import "github.com/fragmede/hn-tui/internal/api"

// FlatComment is a comment flattened from the tree for display.
type FlatComment struct {
	Item        *api.Item
	Depth       int
	IsCollapsed bool
	ChildCount  int
	IsOP        bool
}
