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

// Model is the story detail / comment tree view.
type Model struct {
	viewport    viewport.Model
	story       *api.Item
	comments    []FlatComment
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
			for _, item := range items {
				if item != nil {
					db.PutItem(item)
					// Also pre-fetch one level of nested comments.
					nestedKids := item.Kids()
					if len(nestedKids) > 0 {
						nested, _ := client.BatchGetItems(ctx, nestedKids)
						for _, n := range nested {
							if n != nil {
								db.PutItem(n)
							}
						}
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
	// Reserve space for the story header + hint line.
	m.viewport.Width = w
	m.viewport.Height = h - 6
	if m.viewport.Height < 1 {
		m.viewport.Height = 1
	}
	m.rebuildContent()
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
		m.rebuildComments()
		m.rebuildContent()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selectedIdx < len(m.comments)-1 {
				m.selectedIdx++
				m.rebuildContent()
				m.ensureVisible()
			}
			return m, nil
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.rebuildContent()
				m.ensureVisible()
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
		case "[", "p":
			if idx := FindParentIndex(m.comments, m.selectedIdx); idx >= 0 {
				m.selectedIdx = idx
				m.rebuildContent()
				m.ensureVisible()
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
				m.ensureVisible()
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
		childCount := countDescendants(m.story, m.cache, m.cfg)
		root := FlatComment{
			Item:       m.story,
			Depth:      0,
			ChildCount: childCount,
		}
		m.comments = append([]FlatComment{root}, FlattenTree(m.story.Kids(), m.story.By, m.cache, m.cfg, m.collapse)...)
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
		if m.loading {
			m.viewport.SetContent("  Loading comments...")
		} else {
			m.viewport.SetContent("  No comments yet.")
		}
		return
	}

	var sb strings.Builder
	availWidth := m.width - 4
	if availWidth < 20 {
		availWidth = 20
	}

	for i, fc := range m.comments {
		indent := int(math.Min(float64(fc.Depth*2), 30))
		indentStr := strings.Repeat(" ", indent)

		barColor := depthColors[fc.Depth%len(depthColors)]
		bar := lipgloss.NewStyle().Foreground(barColor).Render("│")

		if fc.Item.Deleted {
			line := indentStr + bar + " " + commentDelStyle.Render("[deleted]")
			sb.WriteString(line + "\n")
			continue
		}
		if fc.Item.Dead {
			line := indentStr + bar + " " + commentDelStyle.Render("[flagged]")
			sb.WriteString(line + "\n")
			continue
		}

		// Header: author + time + collapse indicator.
		header := commentAuthorStyle.Render(fc.Item.By)
		header += " " + commentMetaStyle.Render(render.TimeAgo(fc.Item.Time))
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
		if i == m.selectedIdx {
			headerLine = commentSelStyle.Render(headerLine)
		}
		sb.WriteString(headerLine + "\n")

		if !fc.IsCollapsed {
			for _, line := range strings.Split(body, "\n") {
				bodyLine := indentStr + bar + " " + line
				if i == m.selectedIdx {
					bodyLine = commentSelStyle.Render(bodyLine)
				}
				sb.WriteString(bodyLine + "\n")
			}
		}
		sb.WriteString("\n")
	}

	m.viewport.SetContent(sb.String())
}

func (m *Model) ensureVisible() {
	// Estimate position of selected comment and scroll to it.
	lineCount := 0
	for i, fc := range m.comments {
		if i == m.selectedIdx {
			break
		}
		lineCount += 2 // header + gap
		if !fc.IsCollapsed && fc.Item.Text != "" {
			lineCount += strings.Count(fc.Item.Text, "\n") + 2
		}
	}
	if lineCount < m.viewport.YOffset {
		m.viewport.SetYOffset(lineCount)
	} else if lineCount > m.viewport.YOffset+m.viewport.Height-3 {
		m.viewport.SetYOffset(lineCount - m.viewport.Height/2)
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
	hint := commentMetaStyle.Render("j/k:move  p:parent  ]:sibling  space:collapse  r:reply  e:edit  u:upvote  P:profile")
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
