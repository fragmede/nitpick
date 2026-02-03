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

// feedEntry represents a single item in the threaded comment feed.
type feedEntry struct {
	item  *api.Item
	depth int
}

// feedLoadedMsg is sent when comment feed data is ready.
type feedLoadedMsg struct {
	feedType api.StoryType
	entries  []feedEntry
	err      error
}

type itemOffset struct {
	startLine int
	endLine   int
}

// Model is a viewport-based feed for displaying threaded comments.
// Used for the threads and newcomments tabs.
type Model struct {
	viewport viewport.Model
	entries  []feedEntry
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
	if len(m.entries) > 0 {
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
	m.entries = nil
	m.offsets = nil
	m.viewport.SetContent("")
	m.viewport.SetYOffset(0)
	return m, m.loadFeed()
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case feedLoadedMsg:
		if msg.feedType != m.feedType {
			return m, nil
		}
		if msg.err != nil {
			m.loading = false
			m.viewport.SetContent(errorMsgStyle.Render("Error: " + msg.err.Error()))
			return m, nil
		}
		m.entries = msg.entries
		m.loading = false
		m.cursor = 0
		m.rebuildContent()
		m.viewport.SetYOffset(0)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.entries)-1 {
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
			if m.cursor < len(m.entries) {
				item := m.entries[m.cursor].item
				return m, func() tea.Msg {
					return messages.OpenStoryMsg{StoryID: item.ID}
				}
			}
			return m, nil
		case "o":
			if m.cursor < len(m.entries) {
				item := m.entries[m.cursor].item
				hnURL := fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID)
				return m, func() tea.Msg {
					return messages.StatusMsg{Text: "Opening: " + hnURL}
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
			if len(m.entries) > 0 {
				m.cursor = len(m.entries) - 1
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
	if len(m.entries) == 0 {
		return
	}

	var sb strings.Builder
	m.offsets = make([]itemOffset, len(m.entries))

	contentWidth := m.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	lineCount := 0
	for i, entry := range m.entries {
		startLine := lineCount
		selected := i == m.cursor
		item := entry.item

		indent := entry.depth * 2
		indentStr := strings.Repeat(" ", indent)

		border := normalBorderStyle.Render("▎")
		if selected {
			border = selectedBorderStyle.Render("▎")
		}
		prefix := indentStr + border + " "

		// Meta line first (HN layout: author · time · on: Story Title).
		meta := m.buildMeta(item)
		sb.WriteString(prefix + meta + "\n")
		lineCount++

		// Comment text body.
		bodyWidth := contentWidth - indent - 4
		if bodyWidth < 20 {
			bodyWidth = 20
		}
		text := render.HNToText(item.Text, bodyWidth)
		lines := strings.Split(text, "\n")

		truncated := false
		if len(lines) > maxCommentLines {
			lines = lines[:maxCommentLines]
			truncated = true
		}

		for _, line := range lines {
			sb.WriteString(prefix + line + "\n")
			lineCount++
		}

		if truncated {
			sb.WriteString(prefix + metaStyle.Render("[...]") + "\n")
			lineCount++
		}

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
				return feedLoadedMsg{feedType: st, err: fmt.Errorf("login required for threads")}
			}
			ctx := context.Background()

			// Step 1: Get user's comments from Algolia (has StoryTitle).
			algoliaItems, err := client.GetUserThreads(ctx, username, cfg.FetchPageSize)
			if err != nil {
				return feedLoadedMsg{feedType: st, err: err}
			}

			// Collect IDs and StoryTitles from Algolia results.
			ids := make([]int, 0, len(algoliaItems))
			storyTitles := make(map[int]string)
			for _, item := range algoliaItems {
				if item != nil {
					ids = append(ids, item.ID)
					if item.StoryTitle != "" {
						storyTitles[item.ID] = item.StoryTitle
					}
				}
			}

			// Step 2: Re-fetch from Firebase to get kids arrays.
			fbItems, err := client.BatchGetItems(ctx, ids)
			if err != nil {
				// Fall back to Algolia items without threading.
				entries := make([]feedEntry, 0, len(algoliaItems))
				for _, item := range algoliaItems {
					if item != nil {
						entries = append(entries, feedEntry{item: item, depth: 0})
					}
				}
				return feedLoadedMsg{feedType: st, entries: entries}
			}

			// Merge StoryTitle from Algolia into Firebase items.
			for _, item := range fbItems {
				if item != nil {
					if title, ok := storyTitles[item.ID]; ok {
						item.StoryTitle = title
					}
				}
			}

			// Step 3: Collect all kid IDs for direct replies.
			var allKidIDs []int
			for _, item := range fbItems {
				if item != nil {
					allKidIDs = append(allKidIDs, item.Kids()...)
				}
			}

			// Step 4: Batch-fetch replies.
			replyMap := make(map[int]*api.Item)
			if len(allKidIDs) > 0 {
				replies, _ := client.BatchGetItems(ctx, allKidIDs)
				for _, r := range replies {
					if r != nil {
						replyMap[r.ID] = r
					}
				}
			}

			// Step 5: Build threaded entries (user comment + its replies).
			var entries []feedEntry
			for _, item := range fbItems {
				if item == nil {
					continue
				}
				entries = append(entries, feedEntry{item: item, depth: 0})
				for _, kidID := range item.Kids() {
					if reply, ok := replyMap[kidID]; ok {
						entries = append(entries, feedEntry{item: reply, depth: 1})
					}
				}
			}

			return feedLoadedMsg{feedType: st, entries: entries}
		}

	case api.StoryTypeComments:
		return func() tea.Msg {
			ctx := context.Background()
			items, err := client.GetNewestComments(ctx, cfg.FetchPageSize)
			if err != nil {
				return feedLoadedMsg{feedType: st, err: err}
			}
			entries := make([]feedEntry, 0, len(items))
			for _, item := range items {
				if item != nil {
					entries = append(entries, feedEntry{item: item, depth: 0})
				}
			}
			return feedLoadedMsg{feedType: st, entries: entries}
		}
	}
	return nil
}
