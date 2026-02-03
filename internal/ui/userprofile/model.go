package userprofile

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fragmede/nitpick/internal/api"
	"github.com/fragmede/nitpick/internal/cache"
	"github.com/fragmede/nitpick/internal/config"
	"github.com/fragmede/nitpick/internal/render"
)

var (
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true).Padding(1, 0)
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#828282")).Bold(true)
	valueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	aboutStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC")).Padding(1, 0)
)

type userLoadedMsg struct {
	User *api.User
	Err  error
}

// Model is the user profile view.
type Model struct {
	user     *api.User
	username string
	loading  bool
	err      string
	client   *api.Client
	cache    *cache.DB
	cfg      config.Config
	width    int
	height   int
}

// New creates a new user profile view.
func New(username string, cfg config.Config, client *api.Client, db *cache.DB) Model {
	return Model{
		username: username,
		loading:  true,
		client:   client,
		cache:    db,
		cfg:      cfg,
	}
}

// Init loads the user profile.
func (m Model) Init() tea.Cmd {
	username := m.username
	client := m.client
	db := m.cache
	cfg := m.cfg
	return func() tea.Msg {
		user, fresh, _ := db.GetUser(username, cfg.UserTTL)
		if fresh && user != nil {
			return userLoadedMsg{User: user}
		}
		ctx := context.Background()
		fetched, err := client.GetUser(ctx, username)
		if err != nil {
			if user != nil {
				return userLoadedMsg{User: user}
			}
			return userLoadedMsg{Err: err}
		}
		db.PutUser(fetched)
		return userLoadedMsg{User: fetched}
	}
}

// SetSize sets the viewport dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case userLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err.Error()
		} else {
			m.user = msg.User
		}
	}
	return m, nil
}

// View renders the user profile.
func (m Model) View() string {
	if m.loading {
		return titleStyle.Render("Loading user " + m.username + "...")
	}
	if m.err != "" {
		return titleStyle.Render("Error: " + m.err)
	}
	if m.user == nil {
		return titleStyle.Render("User not found")
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(m.user.ID))
	sb.WriteString("\n")
	sb.WriteString(labelStyle.Render("Karma: ") + valueStyle.Render(fmt.Sprintf("%d", m.user.Karma)))
	sb.WriteString("\n")
	sb.WriteString(labelStyle.Render("Created: ") + valueStyle.Render(render.TimeAgo(m.user.Created)))
	sb.WriteString("\n")

	if m.user.About != "" {
		about := render.HNToText(m.user.About, m.width-4)
		sb.WriteString("\n" + aboutStyle.Render(about))
	}

	return sb.String()
}
