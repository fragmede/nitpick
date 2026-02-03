package storyview

import (
	"github.com/fragmede/hn-tui/internal/api"
	"github.com/fragmede/hn-tui/internal/cache"
	"github.com/fragmede/hn-tui/internal/config"
)

// CollapseState tracks collapsed comment IDs.
type CollapseState map[int]bool

// FlattenTree converts a nested comment tree into a flat list for display.
func FlattenTree(rootKids []int, opUser string, db *cache.DB, cfg config.Config, cs CollapseState) []FlatComment {
	var result []FlatComment

	var walk func(itemID int, depth int)
	walk = func(itemID int, depth int) {
		item, _, _ := db.GetItem(itemID, cfg.CommentTTL)
		if item == nil {
			return
		}

		childCount := countDescendants(item, db, cfg)

		fc := FlatComment{
			Item:        item,
			Depth:       depth,
			IsCollapsed: cs[item.ID],
			ChildCount:  childCount,
			IsOP:        item.By == opUser && opUser != "",
		}
		result = append(result, fc)

		if cs[item.ID] {
			return
		}

		for _, kidID := range item.Kids() {
			walk(kidID, depth+1)
		}
	}

	for _, kidID := range rootKids {
		walk(kidID, 0)
	}
	return result
}

func countDescendants(item *api.Item, db *cache.DB, cfg config.Config) int {
	count := 0
	for _, kidID := range item.Kids() {
		count++
		kid, _, _ := db.GetItem(kidID, cfg.CommentTTL)
		if kid != nil {
			count += countDescendants(kid, db, cfg)
		}
	}
	return count
}

// FindParentIndex returns the index of the parent comment in the flat list.
func FindParentIndex(comments []FlatComment, currentIdx int) int {
	if currentIdx < 0 || currentIdx >= len(comments) {
		return -1
	}
	parentID := comments[currentIdx].Item.Parent
	for i := currentIdx - 1; i >= 0; i-- {
		if comments[i].Item.ID == parentID {
			return i
		}
	}
	return -1
}

// FindNextSiblingIndex returns the index of the next comment at the same depth.
func FindNextSiblingIndex(comments []FlatComment, currentIdx int) int {
	if currentIdx < 0 || currentIdx >= len(comments) {
		return -1
	}
	depth := comments[currentIdx].Depth
	for i := currentIdx + 1; i < len(comments); i++ {
		if comments[i].Depth < depth {
			return -1 // Went up in tree, no more siblings.
		}
		if comments[i].Depth == depth {
			return i
		}
	}
	return -1
}
