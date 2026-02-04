package storyview

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fragmede/nitpick/internal/api"
	"github.com/fragmede/nitpick/internal/cache"
	"github.com/fragmede/nitpick/internal/config"
	"github.com/fragmede/nitpick/internal/render"
	"github.com/fragmede/nitpick/internal/ui/messages"
)

var (
	depthColors = []lipgloss.Color{
		"#FF6600", "#828282", "#00BFFF", "#32CD32", "#FFD700", "#FF69B4", "#9370DB", "#20B2AA",
	}

	commentAuthorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true)
	commentMetaStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	commentOPStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#000")).Background(lipgloss.Color("#FF6600")).Bold(true)
	commentSelStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#333333"))
	commentDelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Italic(true)
	storyHeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1)
	storyMetaStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#828282")).Padding(0, 1)
	storyURLStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#828282")).Padding(0, 1)
	separatorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)

const scrollStep = 3

type commentOffset struct {
	startLine int
	endLine   int
}

// Model is the story detail / comment tree view.
type Model struct {
	viewport    viewport.Model
	story       *api.Item
	comments    []FlatComment
	offsets     []commentOffset
	selectedIdx int
	collapse    CollapseState
	client      *api.Client
	cache       *cache.DB
	cfg         config.Config
	username    string
	loading     bool
	width       int
	height      int
}

// New creates a new story view.
func New(storyID int, cfg config.Config, client *api.Client, db *cache.DB, username string) Model {
	vp := viewport.New(0, 0)
	vp.SetContent("Loading...")

	return Model{
		viewport:    vp,
		collapse:    make(CollapseState),
		client:      client,
		cache:       db,
		cfg:         cfg,
		username:    username,
		loading:     true,
		selectedIdx: 0,
	}
}

// Init loads the story and its comments.
func (m Model) Init(storyID int) tea.Cmd {
	client := m.client
	db := m.cache
	cfg := m.cfg
	return func() tea.Msg {
		ctx := context.Background()
		story, _, _ := db.GetItem(storyID, cfg.ItemTTL)
		if story == nil {
			var err error
			story, err = client.GetItem(ctx, storyID)
			if err != nil {
				return messages.CommentsLoadedMsg{StoryID: storyID, Err: err}
			}
			db.PutItem(story)
		}

		// Fetch top-level comments.
		kids := story.Kids()
		if len(kids) > 0 {
			items, _ := client.BatchGetItems(ctx, kids)
			// Collect all nested kid IDs, then fetch in one batch.
			var allNestedIDs []int
			for _, item := range items {
				if item != nil {
					db.PutItem(item)
					allNestedIDs = append(allNestedIDs, item.Kids()...)
				}
			}
			if len(allNestedIDs) > 0 {
				nested, _ := client.BatchGetItems(ctx, allNestedIDs)
				for _, n := range nested {
					if n != nil {
						db.PutItem(n)
					}
				}
			}
		}

		return messages.CommentsLoadedMsg{StoryID: storyID, Items: []*api.Item{story}}
	}
}

// SetSize updates viewport dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.resizeViewport()
	m.rebuildContent()
}

func (m *Model) resizeViewport() {
	header := m.renderHeader()
	headerLines := strings.Count(header, "\n") + 1
	m.viewport.Height = m.height - headerLines
	if m.viewport.Height < 1 {
		m.viewport.Height = 1
	}
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.CommentsLoadedMsg:
		if msg.Err != nil {
			m.viewport.SetContent("Error loading comments: " + msg.Err.Error())
			m.loading = false
			return m, nil
		}
		if len(msg.Items) > 0 {
			m.story = msg.Items[0]
		}
		m.loading = false
		m.resizeViewport()
		m.rebuildComments()
		m.rebuildContent()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.offsets) {
				off := m.offsets[m.selectedIdx]
				viewBottom := m.viewport.YOffset + m.viewport.Height
				if off.endLine >= viewBottom {
					// Comment extends below viewport — scroll within it.
					m.viewport.SetYOffset(m.viewport.YOffset + scrollStep)
					return m, nil
				}
			}
			if m.selectedIdx < len(m.comments)-1 {
				m.selectedIdx++
				m.rebuildContent()
				m.scrollToCursor()
			}
			return m, nil
		case "k", "up":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.offsets) {
				off := m.offsets[m.selectedIdx]
				if off.startLine < m.viewport.YOffset {
					// Comment extends above viewport — scroll within it.
					newOff := m.viewport.YOffset - scrollStep
					if newOff < off.startLine {
						newOff = off.startLine
					}
					m.viewport.SetYOffset(newOff)
					return m, nil
				}
			}
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.rebuildContent()
				m.scrollToCursor()
			}
			return m, nil
		case "enter", " ":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.comments) {
				id := m.comments[m.selectedIdx].Item.ID
				m.collapse[id] = !m.collapse[id]
				m.rebuildComments()
				m.rebuildContent()
			}
			return m, nil
		case "z":
			// Toggle collapse all: if any are expanded, collapse all; otherwise expand all.
			anyExpanded := false
			for _, fc := range m.comments {
				if !m.collapse[fc.Item.ID] && len(fc.Item.Kids()) > 0 {
					anyExpanded = true
					break
				}
			}
			for _, fc := range m.comments {
				if len(fc.Item.Kids()) > 0 {
					m.collapse[fc.Item.ID] = anyExpanded
				}
			}
			m.rebuildComments()
			m.rebuildContent()
			if anyExpanded {
				m.viewport.GotoTop()
				m.selectedIdx = 0
			}
			return m, nil
		case "[", "p":
			if idx := FindParentIndex(m.comments, m.selectedIdx); idx >= 0 {
				m.selectedIdx = idx
				m.rebuildContent()
				m.scrollToCursor()
			} else if m.selectedIdx >= 0 && m.selectedIdx < len(m.comments) {
				// Parent not in current view — navigate to the parent item.
				parentID := m.comments[m.selectedIdx].Item.Parent
				if parentID > 0 {
					return m, func() tea.Msg {
						return messages.OpenStoryMsg{StoryID: parentID}
					}
				}
			}
			return m, nil
		case "]":
			if idx := FindNextSiblingIndex(m.comments, m.selectedIdx); idx >= 0 {
				m.selectedIdx = idx
				m.rebuildContent()
				m.scrollToCursor()
			}
			return m, nil
		case "g", "home":
			m.selectedIdx = 0
			m.rebuildContent()
			m.viewport.GotoTop()
			return m, nil
		case "G", "end":
			if len(m.comments) > 0 {
				m.selectedIdx = len(m.comments) - 1
				m.rebuildContent()
				m.viewport.GotoBottom()
			}
			return m, nil
		case "r":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.comments) {
				parentID := m.comments[m.selectedIdx].Item.ID
				return m, func() tea.Msg { return messages.OpenReplyMsg{ParentID: parentID} }
			}
			return m, nil
		case "R":
			if m.story == nil || m.story.Parent == 0 {
				return m, nil // already at root
			}
			db := m.cache
			cfg := m.cfg
			current := m.story
			for current.Parent != 0 {
				parent, _, _ := db.GetItem(current.Parent, cfg.ItemTTL)
				if parent == nil {
					// Parent not cached; open whatever we know is above us.
					rootID := current.Parent
					return m, func() tea.Msg { return messages.OpenStoryMsg{StoryID: rootID} }
				}
				if parent.Type == "story" {
					rootID := parent.ID
					return m, func() tea.Msg { return messages.OpenStoryMsg{StoryID: rootID} }
				}
				current = parent
			}
			rootID := current.ID
			return m, func() tea.Msg { return messages.OpenStoryMsg{StoryID: rootID} }
		case "ctrl+r":
			if m.story != nil {
				m.loading = true
				m.cache.InvalidateItem(m.story.ID)
				m.viewport.SetContent("  Refreshing...")
				return m, m.Init(m.story.ID)
			}
			return m, nil
		case "o":
			if m.story != nil && m.story.URL != "" {
				return m, openURL(m.story.URL)
			}
			return m, nil
		case "ctrl+d", "pgdown":
			m.viewport.HalfViewDown()
			return m, nil
		case "ctrl+u", "pgup":
			m.viewport.HalfViewUp()
			return m, nil
		case "P":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.comments) {
				username := m.comments[m.selectedIdx].Item.By
				if username != "" {
					return m, func() tea.Msg { return messages.OpenUserMsg{Username: username} }
				}
			}
			return m, nil
		case "e":
			if m.selectedIdx < 0 || m.selectedIdx >= len(m.comments) {
				return m, nil
			}
			item := m.comments[m.selectedIdx].Item
			if m.username == "" {
				return m, func() tea.Msg {
					return messages.StatusMsg{Text: "Login required to edit"}
				}
			}
			if item.By != m.username {
				return m, func() tea.Msg {
					return messages.StatusMsg{Text: "Can only edit your own comments"}
				}
			}
			elapsed := time.Now().Unix() - item.Time
			if elapsed >= 7200 {
				return m, func() tea.Msg {
					return messages.StatusMsg{Text: "Edit window has expired (2 hour limit)"}
				}
			}
			return m, func() tea.Msg {
				return messages.OpenEditMsg{ItemID: item.ID, CurrentText: item.Text}
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the story view.
func (m Model) View() string {
	header := m.renderHeader()
	return lipgloss.JoinVertical(lipgloss.Left, header, m.viewport.View())
}

// Story returns the current story item.
func (m Model) Story() *api.Item {
	return m.story
}

func (m *Model) rebuildComments() {
	if m.story == nil {
		m.comments = nil
		return
	}
	if m.story.Type == "comment" {
		// Include the root comment itself as the first selectable item.
		kids := FlattenTree(m.story.Kids(), m.story.By, m.cache, m.cfg, m.collapse)
		root := FlatComment{
			Item:       m.story,
			Depth:      0,
			ChildCount: len(kids),
		}
		m.comments = append([]FlatComment{root}, kids...)
	} else {
		m.comments = FlattenTree(m.story.Kids(), m.story.By, m.cache, m.cfg, m.collapse)
	}
	if m.selectedIdx >= len(m.comments) {
		m.selectedIdx = len(m.comments) - 1
	}
	if m.selectedIdx < 0 {
		m.selectedIdx = 0
	}
}

func (m *Model) rebuildContent() {
	if len(m.comments) == 0 {
		m.offsets = nil
		if m.loading {
			m.viewport.SetContent("  Loading comments...")
		} else {
			m.viewport.SetContent("  No comments yet.")
		}
		return
	}

	var sb strings.Builder
	m.offsets = make([]commentOffset, len(m.comments))
	availWidth := m.width - 4
	if availWidth < 20 {
		availWidth = 20
	}

	lineCount := 0
	for i, fc := range m.comments {
		startLine := lineCount
		indent := int(math.Min(float64(fc.Depth*2), 30))
		indentStr := strings.Repeat(" ", indent)

		barColor := depthColors[fc.Depth%len(depthColors)]
		selected := i == m.selectedIdx
		if selected {
			barColor = "#FF6600"
		}
		bar := lipgloss.NewStyle().Foreground(barColor).Render("│")

		if fc.Item.Deleted {
			line := indentStr + bar + " " + commentDelStyle.Render("[deleted]")
			sb.WriteString(line + "\n")
			lineCount++
			m.offsets[i] = commentOffset{startLine: startLine, endLine: lineCount - 1}
			continue
		}
		if fc.Item.Dead {
			line := indentStr + bar + " " + commentDelStyle.Render("[flagged]")
			sb.WriteString(line + "\n")
			lineCount++
			m.offsets[i] = commentOffset{startLine: startLine, endLine: lineCount - 1}
			continue
		}

		// Header: author + time + score + collapse indicator.
		header := commentAuthorStyle.Render(fc.Item.By)
		header += " " + commentMetaStyle.Render(render.TimeAgo(fc.Item.Time))
		if fc.Item.Score > 0 {
			header += " " + commentMetaStyle.Render(fmt.Sprintf("%d points", fc.Item.Score))
		}
		if fc.IsOP {
			header += " " + commentOPStyle.Render(" OP ")
		}
		if fc.IsCollapsed {
			header += " " + commentMetaStyle.Render(fmt.Sprintf("[+%d]", fc.ChildCount))
		}
		if fc.Depth > 15 {
			header += " " + commentMetaStyle.Render(fmt.Sprintf("[d:%d]", fc.Depth))
		}

		// Body text.
		bodyWidth := availWidth - indent - 4
		if bodyWidth < 20 {
			bodyWidth = 20
		}
		body := render.HNToText(fc.Item.Text, bodyWidth)

		// Compose lines.
		headerLine := indentStr + bar + " " + header
		if selected {
			headerLine = commentSelStyle.Render(headerLine)
		}
		sb.WriteString(headerLine + "\n")
		lineCount++

		if !fc.IsCollapsed {
			for _, line := range strings.Split(body, "\n") {
				bodyLine := indentStr + bar + " " + line
				if selected {
					bodyLine = commentSelStyle.Render(bodyLine)
				}
				sb.WriteString(bodyLine + "\n")
				lineCount++
			}
		}
		sb.WriteString("\n")
		lineCount++

		m.offsets[i] = commentOffset{startLine: startLine, endLine: lineCount - 1}
	}

	m.viewport.SetContent(sb.String())
}

func (m *Model) scrollToCursor() {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.offsets) {
		return
	}
	off := m.offsets[m.selectedIdx]
	// Show the start of the selected comment if it's not already visible.
	if off.startLine < m.viewport.YOffset || off.startLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(off.startLine)
	}
}

func (m Model) renderHeader() string {
	if m.story == nil {
		return storyHeaderStyle.Render("Loading...")
	}

	var parts []string

	if m.story.Title != "" {
		// Story header.
		parts = append(parts, storyHeaderStyle.Render(m.story.Title))
		parts = append(parts, storyMetaStyle.Render(fmt.Sprintf(
			"%d points | by %s | %s | %d comments",
			m.story.Score, m.story.By, render.TimeAgo(m.story.Time), m.story.Descendants,
		)))
		if m.story.URL != "" {
			if u, err := url.Parse(m.story.URL); err == nil {
				parts = append(parts, storyURLStyle.Render(u.Host))
			}
		}
		if m.story.Text != "" {
			bodyWidth := m.width - 4
			if bodyWidth < 20 {
				bodyWidth = 20
			}
			parts = append(parts, storyMetaStyle.Render(render.HNToText(m.story.Text, bodyWidth)))
		}
	} else if m.story.Type == "comment" {
		// Comment root — minimal header since the full text is in the list.
		meta := fmt.Sprintf("Comment by %s | %s", m.story.By, render.TimeAgo(m.story.Time))
		kids := m.story.Kids()
		if len(kids) > 0 {
			meta += fmt.Sprintf(" | %d replies", len(kids))
		}
		parts = append(parts, storyMetaStyle.Render(meta))
	} else {
		parts = append(parts, storyHeaderStyle.Render(fmt.Sprintf("[%s #%d]", m.story.Type, m.story.ID)))
	}

	parts = append(parts, separatorStyle.Render(strings.Repeat("─", m.width)))
	hint := commentMetaStyle.Render("j/k:move  p:parent  ]:sibling  R:root  space:collapse  z:fold all  r:reply  e:edit  P:profile")
	parts = append(parts, hint)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func openURL(u string) tea.Cmd {
	return func() tea.Msg {
		// Use 'open' on macOS, 'xdg-open' on Linux.
		// This runs in a goroutine via tea.Cmd.
		return messages.StatusMsg{Text: "Opening: " + u}
	}
}
