package commentfeed

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fragmede/nitpick/internal/api"
	"github.com/fragmede/nitpick/internal/config"
	"github.com/fragmede/nitpick/internal/render"
	"github.com/fragmede/nitpick/internal/ui/messages"
)

const maxCommentLines = 20

var (
	selectedBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600"))
	normalBorderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	authorStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true)
	metaStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#828282"))
	storyRefStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	headerStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true)
	errorMsgStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
)

type itemOffset struct {
	startLine int
	endLine   int
}

// Model is a viewport-based feed for displaying full comment text.
// Used for the threads and newcomments tabs.
type Model struct {
	viewport viewport.Model
	items    []*api.Item
	offsets  []itemOffset
	cursor   int
	feedType api.StoryType
	client   *api.Client
	cfg      config.Config
	username string
	loading  bool
	width    int
	height   int
}

// New creates a new comment feed model.
func New(cfg config.Config, client *api.Client) Model {
	vp := viewport.New(0, 0)
	return Model{
		viewport: vp,
		client:   client,
		cfg:      cfg,
		feedType: api.StoryTypeComments,
	}
}

// SetSize updates viewport dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h - 2 // title + blank line
	if m.viewport.Height < 1 {
		m.viewport.Height = 1
	}
	if len(m.items) > 0 {
		m.rebuildContent()
	}
}

// SetUser sets the logged-in username (needed for threads tab).
func (m *Model) SetUser(username string) {
	m.username = username
}

// FeedType returns the current feed type.
func (m Model) FeedType() api.StoryType {
	return m.feedType
}

// SwitchFeed changes the feed type and triggers a load.
func (m Model) SwitchFeed(st api.StoryType) (Model, tea.Cmd) {
	m.feedType = st
	m.loading = true
	m.cursor = 0
	m.items = nil
	m.offsets = nil
	m.viewport.SetContent("")
	m.viewport.SetYOffset(0)
	return m, m.loadFeed()
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.StoriesLoadedMsg:
		if msg.StoryType != m.feedType {
			return m, nil
		}
		if msg.Err != nil {
			m.loading = false
			m.viewport.SetContent(errorMsgStyle.Render("Error: " + msg.Err.Error()))
			return m, nil
		}
		m.items = make([]*api.Item, 0, len(msg.Items))
		for _, item := range msg.Items {
			if item != nil {
				m.items = append(m.items, item)
			}
		}
		m.loading = false
		m.cursor = 0
		m.rebuildContent()
		m.viewport.SetYOffset(0)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
				m.rebuildContent()
				m.scrollToCursor()
			}
			return m, nil
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.rebuildContent()
				m.scrollToCursor()
			}
			return m, nil
		case "enter":
			if m.cursor < len(m.items) {
				item := m.items[m.cursor]
				return m, func() tea.Msg {
					return messages.OpenStoryMsg{StoryID: item.ID}
				}
			}
			return m, nil
		case "r":
			m.loading = true
			m.viewport.SetContent("Refreshing...")
			return m, m.loadFeed()
		case "g", "home":
			m.cursor = 0
			m.rebuildContent()
			m.viewport.GotoTop()
			return m, nil
		case "G", "end":
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
				m.rebuildContent()
				m.viewport.GotoBottom()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the comment feed.
func (m Model) View() string {
	title := m.title()
	if m.loading {
		title += " (loading...)"
	}
	header := headerStyle.Render(title) + "\n"
	return header + m.viewport.View()
}

func (m *Model) rebuildContent() {
	if len(m.items) == 0 {
		return
	}

	var sb strings.Builder
	m.offsets = make([]itemOffset, len(m.items))

	contentWidth := m.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	lineCount := 0
	for i, item := range m.items {
		startLine := lineCount
		selected := i == m.cursor

		border := normalBorderStyle.Render("▎")
		if selected {
			border = selectedBorderStyle.Render("▎")
		}

		// Comment text body.
		text := render.HNToText(item.Text, contentWidth)
		lines := strings.Split(text, "\n")

		truncated := false
		if len(lines) > maxCommentLines {
			lines = lines[:maxCommentLines]
			truncated = true
		}

		for _, line := range lines {
			sb.WriteString(border + " " + line + "\n")
			lineCount++
		}

		if truncated {
			sb.WriteString(border + " " + metaStyle.Render("[...]") + "\n")
			lineCount++
		}

		// Metadata line: author · time · on: Story Title
		meta := m.buildMeta(item)
		sb.WriteString(border + " " + meta + "\n")
		lineCount++

		// Blank separator.
		sb.WriteString("\n")
		lineCount++

		m.offsets[i] = itemOffset{startLine: startLine, endLine: lineCount - 1}
	}

	m.viewport.SetContent(sb.String())
}

func (m *Model) buildMeta(item *api.Item) string {
	var parts []string
	sep := metaStyle.Render(" · ")

	if item.By != "" {
		parts = append(parts, authorStyle.Render(item.By))
	}
	parts = append(parts, metaStyle.Render(render.TimeAgo(item.Time)))

	if item.StoryTitle != "" {
		parts = append(parts, metaStyle.Render("on: ")+storyRefStyle.Render(item.StoryTitle))
	}

	return strings.Join(parts, sep)
}

func (m *Model) scrollToCursor() {
	if m.cursor >= len(m.offsets) {
		return
	}
	ri := m.offsets[m.cursor]

	if ri.startLine < m.viewport.YOffset {
		m.viewport.SetYOffset(ri.startLine)
	}
	if ri.endLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(ri.startLine)
	}
}

func (m Model) title() string {
	switch m.feedType {
	case api.StoryTypeThreads:
		if m.username != "" {
			return "Threads — " + m.username
		}
		return "My Threads"
	case api.StoryTypeComments:
		return "New Comments"
	default:
		return "Comments"
	}
}

func (m Model) loadFeed() tea.Cmd {
	st := m.feedType
	client := m.client
	cfg := m.cfg
	username := m.username

	switch st {
	case api.StoryTypeThreads:
		return func() tea.Msg {
			if username == "" {
				return messages.StoriesLoadedMsg{StoryType: st, Err: fmt.Errorf("login required for threads")}
			}
			items, err := client.GetUserThreads(context.Background(), username, cfg.FetchPageSize)
			return messages.StoriesLoadedMsg{StoryType: st, Items: items, Err: err}
		}
	case api.StoryTypeComments:
		return func() tea.Msg {
			items, err := client.GetNewestComments(context.Background(), cfg.FetchPageSize)
			return messages.StoriesLoadedMsg{StoryType: st, Items: items, Err: err}
		}
	}
	return nil
}
