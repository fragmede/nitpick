package statusbar

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fragmede/hn-tui/internal/api"
)

var (
	barStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#FFFFFF"))

	activeTabStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FF6600")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#555555")).
				Foreground(lipgloss.Color("#CCCCCC")).
				Padding(0, 1)

	userStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#00FF00")).
			Padding(0, 1)

	notifyStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FF0000")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 1)

	statusTextStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#AAAAAA")).
			Padding(0, 1)

	offlineStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#8B0000")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 1)
)

type tab struct {
	label     string
	storyType api.StoryType
}

var tabs = []tab{
	{"Top", api.StoryTypeTop},
	{"New", api.StoryTypeNew},
	{"Threads", api.StoryTypeThreads},
	{"Past", api.StoryTypePast},
	{"Comments", api.StoryTypeComments},
	{"Ask", api.StoryTypeAsk},
	{"Show", api.StoryTypeShow},
	{"Jobs", api.StoryTypeJobs},
}

// Model is the status bar at the bottom of the screen.
type Model struct {
	width       int
	activeType  api.StoryType
	username    string
	unreadCount int
	statusText  string
	offline     bool
}

// New creates a new status bar.
func New() Model {
	return Model{activeType: api.StoryTypeTop}
}

// SetSize sets the width.
func (m *Model) SetSize(w int) {
	m.width = w
}

// SetActiveTab sets the active story type tab.
func (m *Model) SetActiveTab(st api.StoryType) {
	m.activeType = st
}

// SetUser sets the logged-in username.
func (m *Model) SetUser(username string) {
	m.username = username
}

// SetUnread sets the unread notification count.
func (m *Model) SetUnread(count int) {
	m.unreadCount = count
}

// SetStatus sets a temporary status message.
func (m *Model) SetStatus(text string) {
	m.statusText = text
}

// SetOffline sets the offline indicator.
func (m *Model) SetOffline(offline bool) {
	m.offline = offline
}

// Update is a no-op for the status bar.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

// View renders the status bar.
func (m Model) View() string {
	// Tabs.
	var tabsStr string
	for _, t := range tabs {
		if t.storyType == m.activeType {
			tabsStr += activeTabStyle.Render(t.label)
		} else {
			tabsStr += inactiveTabStyle.Render(t.label)
		}
	}

	// Right side.
	var right string
	if m.offline {
		right += offlineStyle.Render("OFFLINE")
	}
	if m.unreadCount > 0 {
		right += notifyStyle.Render(fmt.Sprintf(" %d ", m.unreadCount))
	}
	if m.username != "" {
		right += userStyle.Render(m.username)
	} else {
		right += statusTextStyle.Render("L:login")
	}
	if m.statusText != "" {
		right += statusTextStyle.Render(m.statusText)
	}

	// Fill middle with background.
	tabsWidth := lipgloss.Width(tabsStr)
	rightWidth := lipgloss.Width(right)
	gap := m.width - tabsWidth - rightWidth
	if gap < 0 {
		gap = 0
	}
	mid := barStyle.Width(gap).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Top, tabsStr, mid, right)
}
