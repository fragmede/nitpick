package login

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fragmede/hn-tui/internal/auth"
	"github.com/fragmede/hn-tui/internal/ui/messages"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600"))
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true).
			Padding(1, 0)
)

// Model is the login form view.
type Model struct {
	usernameInput textinput.Model
	passwordInput textinput.Model
	focusIndex    int
	err           string
	submitting    bool
	session       *auth.Session
	width         int
	height        int
}

// New creates a new login form.
func New(session *auth.Session) Model {
	usernameInput := textinput.New()
	usernameInput.Placeholder = "username"
	usernameInput.Focus()
	usernameInput.Width = 30

	passwordInput := textinput.New()
	passwordInput.Placeholder = "password"
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.Width = 30

	return Model{
		usernameInput: usernameInput,
		passwordInput: passwordInput,
		session:       session,
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
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab":
			if m.focusIndex == 0 {
				m.focusIndex = 1
				m.usernameInput.Blur()
				m.passwordInput.Focus()
			} else {
				m.focusIndex = 0
				m.passwordInput.Blur()
				m.usernameInput.Focus()
			}
			return m, nil
		case "enter":
			if m.submitting {
				return m, nil
			}
			username := strings.TrimSpace(m.usernameInput.Value())
			password := m.passwordInput.Value()
			if username == "" || password == "" {
				m.err = "Username and password required"
				return m, nil
			}
			m.submitting = true
			m.err = ""
			session := m.session
			return m, func() tea.Msg {
				err := session.Login(username, password)
				if err != nil {
					return messages.LoginResultMsg{Err: err}
				}
				return messages.LoginResultMsg{Username: username}
			}
		}

	case messages.LoginResultMsg:
		m.submitting = false
		if msg.Err != nil {
			m.err = msg.Err.Error()
			return m, nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.focusIndex == 0 {
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	} else {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	return m, cmd
}

// View renders the login form.
func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Login to Hacker News"))
	sb.WriteString("\n\n")
	sb.WriteString(labelStyle.Render("Username:"))
	sb.WriteString("\n")
	sb.WriteString(m.usernameInput.View())
	sb.WriteString("\n\n")
	sb.WriteString(labelStyle.Render("Password:"))
	sb.WriteString("\n")
	sb.WriteString(m.passwordInput.View())
	sb.WriteString("\n\n")

	if m.err != "" {
		sb.WriteString(errorStyle.Render(m.err))
		sb.WriteString("\n\n")
	}

	if m.submitting {
		sb.WriteString("Logging in...")
	} else {
		sb.WriteString(focusedStyle.Render("Enter") + " to submit, " + focusedStyle.Render("Esc") + " to cancel")
	}

	content := sb.String()
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
