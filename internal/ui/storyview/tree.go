package storyview

import (
	"github.com/fragmede/nitpick/internal/cache"
	"github.com/fragmede/nitpick/internal/config"
)

// CollapseState tracks collapsed comment IDs.
type CollapseState map[int]bool

// FlattenTree converts a nested comment tree into a flat list for display.
func FlattenTree(rootKids []int, opUser string, db *cache.DB, cfg config.Config, cs CollapseState) []FlatComment {
	var result []FlatComment

	// walk returns the total descendant count for this subtree.
	var walk func(itemID int, depth int) int
	walk = func(itemID int, depth int) int {
		item, _, _ := db.GetItem(itemID, cfg.CommentTTL)
		if item == nil {
			return 0
		}

		idx := len(result)
		// Append placeholder; we'll fill ChildCount after walking children.
		result = append(result, FlatComment{
			Item:        item,
			Depth:       depth,
			IsCollapsed: cs[item.ID],
			IsOP:        item.By == opUser && opUser != "",
		})

		descendants := 0
		if !cs[item.ID] {
			for _, kidID := range item.Kids() {
				descendants += 1 + walk(kidID, depth+1)
			}
		} else {
			// Collapsed: count kids without walking (for the [+N] badge).
			descendants = len(item.Kids())
		}

		result[idx].ChildCount = descendants
		return descendants
	}

	for _, kidID := range rootKids {
		walk(kidID, 0)
	}
	return result
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
