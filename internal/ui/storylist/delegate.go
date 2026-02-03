package storylist

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	indexStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6600")).
			Width(4).
			Align(lipgloss.Right)

	titleNormal = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF"))

	titleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6600"))

	domainStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#828282"))

	metaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#828282"))

	commentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6600"))

	commentSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6600")).
				Bold(true).
				Underline(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))
)

type Delegate struct{}

func (d Delegate) Height() int                             { return 2 }
func (d Delegate) Spacing() int                            { return 1 }
func (d Delegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d Delegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(StoryItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	idx := indexStyle.Render(fmt.Sprintf("%d.", item.Index+1))

	if item.IsComment() {
		d.renderComment(w, idx, item, selected)
	} else {
		d.renderStory(w, idx, item, selected)
	}
}

func (d Delegate) renderStory(w io.Writer, idx string, item StoryItem, selected bool) {
	// Line 1: index. Title (domain)
	var title string
	if selected {
		title = titleSelected.Render(item.Title())
	} else {
		title = titleNormal.Render(item.Title())
	}

	domain := item.Domain()
	if domain != "" {
		title += " " + domainStyle.Render("("+domain+")")
	}

	// Line 2: N points by author | time | N comments
	var meta string
	if item.Item.Score > 0 {
		meta += fmt.Sprintf("%d points ", item.Item.Score)
	}
	if item.Item.By != "" {
		meta += fmt.Sprintf("by %s ", item.Item.By)
	}
	meta += item.TimeAgo()

	metaStr := metaStyle.Render(meta)

	// Comment count — styled separately to stand out like on HN.
	comments := item.CommentStr()
	if comments != "" {
		sep := separatorStyle.Render(" | ")
		if selected {
			metaStr += sep + commentSelectedStyle.Render(comments)
		} else {
			metaStr += sep + commentStyle.Render(comments)
		}
	}

	fmt.Fprintf(w, "%s %s\n     %s", idx, title, metaStr)
}

func (d Delegate) renderComment(w io.Writer, idx string, item StoryItem, selected bool) {
	// Line 1: index. Comment text preview
	var title string
	if selected {
		title = titleSelected.Render(item.Title())
	} else {
		title = titleNormal.Render(item.Title())
	}

	// Line 2: by author | time | on: Parent Story Title
	meta := ""
	if item.Item.By != "" {
		meta += fmt.Sprintf("by %s ", item.Item.By)
	}
	meta += item.TimeAgo()
	metaStr := metaStyle.Render(meta)

	if item.Item.StoryTitle != "" {
		sep := separatorStyle.Render(" | ")
		metaStr += sep + metaStyle.Render("on: ") + commentStyle.Render(item.Item.StoryTitle)
	}

	fmt.Fprintf(w, "%s %s\n     %s", idx, title, metaStr)
}
