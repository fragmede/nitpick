package storylist

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF"))

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#828282"))

	selectedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF6600"))

	selectedDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CCCCCC"))

	indexStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6600")).
			Width(4).
			Align(lipgloss.Right)
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

	idx := indexStyle.Render(fmt.Sprintf("%d.", item.Index+1))

	var title, desc string
	if index == m.Index() {
		title = selectedTitleStyle.Render(item.Title())
		desc = selectedDescStyle.Render(item.Description())
	} else {
		title = titleStyle.Render(item.Title())
		desc = descStyle.Render(item.Description())
	}

	fmt.Fprintf(w, "%s %s\n   %s", idx, title, desc)
}
