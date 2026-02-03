package notifications

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fragmede/hn-tui/internal/cache"
	"github.com/fragmede/hn-tui/internal/render"
	"github.com/fragmede/hn-tui/internal/ui/messages"
)

var (
	titleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true).Padding(1, 0)
	notifStyle     = lipgloss.NewStyle().Padding(0, 1)
	selectedStyle  = lipgloss.NewStyle().Background(lipgloss.Color("#333333")).Padding(0, 1)
	authorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true)
	unreadDotStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	metaStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	previewStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
)

// Notification represents a single notification entry.
type Notification struct {
	ID          int
	ItemID      int
	ParentID    int
	StoryID     int
	ByUser      string
	TextPreview string
	CreatedAt   int64
	Read        bool
}

// Model is the notifications view.
type Model struct {
	notifications []Notification
	selectedIdx   int
	db            *cache.DB
	width         int
	height        int
}

// New creates a new notifications model.
func New(db *cache.DB) Model {
	return Model{db: db}
}

// SetSize sets the viewport dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Load refreshes the notification list from the database.
func (m *Model) Load() {
	m.notifications = loadNotifications(m.db)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selectedIdx < len(m.notifications)-1 {
				m.selectedIdx++
			}
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "enter":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.notifications) {
				n := m.notifications[m.selectedIdx]
				markRead(m.db, n.ID)
				m.notifications[m.selectedIdx].Read = true
				return m, func() tea.Msg {
					return messages.OpenStoryMsg{StoryID: n.StoryID}
				}
			}
		}
	}
	return m, nil
}

// View renders the notifications list.
func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Notifications"))
	sb.WriteString("\n")

	if len(m.notifications) == 0 {
		sb.WriteString("\n  No notifications yet.\n")
		return sb.String()
	}

	for i, n := range m.notifications {
		var line strings.Builder

		if !n.Read {
			line.WriteString(unreadDotStyle.Render("● "))
		} else {
			line.WriteString("  ")
		}

		line.WriteString(authorStyle.Render(n.ByUser))
		line.WriteString(metaStyle.Render(fmt.Sprintf(" replied %s", render.TimeAgo(n.CreatedAt))))
		line.WriteString("\n")
		if n.TextPreview != "" {
			preview := n.TextPreview
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			line.WriteString("  " + previewStyle.Render(preview))
		}

		entry := line.String()
		if i == m.selectedIdx {
			entry = selectedStyle.Render(entry)
		} else {
			entry = notifStyle.Render(entry)
		}
		sb.WriteString(entry + "\n")
	}

	return sb.String()
}

// UnreadCount returns the number of unread notifications.
func (m Model) UnreadCount() int {
	count := 0
	for _, n := range m.notifications {
		if !n.Read {
			count++
		}
	}
	return count
}

func loadNotifications(db *cache.DB) []Notification {
	rows, err := db.Query(`SELECT id, item_id, parent_id, story_id, by_user, text_preview, created_at, read
		FROM notifications ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []Notification
	for rows.Next() {
		var n Notification
		var readInt int
		if err := rows.Scan(&n.ID, &n.ItemID, &n.ParentID, &n.StoryID, &n.ByUser,
			&n.TextPreview, &n.CreatedAt, &readInt); err != nil {
			continue
		}
		n.Read = readInt != 0
		result = append(result, n)
	}
	return result
}

func markRead(db *cache.DB, id int) {
	db.Exec("UPDATE notifications SET read = 1 WHERE id = ?", id)
}
