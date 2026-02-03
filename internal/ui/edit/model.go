package edit

import (
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fragmede/nitpick/internal/auth"
	"github.com/fragmede/nitpick/internal/render"
	"github.com/fragmede/nitpick/internal/ui/messages"
)

var (
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true)
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#828282"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
)

// Model is the comment edit view.
type Model struct {
	textarea   textarea.Model
	itemID     int
	session    *auth.Session
	err        string
	submitting bool
	width      int
	height     int
}

// New creates a new edit form pre-filled with the current comment text.
func New(itemID int, currentText string, session *auth.Session) Model {
	ta := textarea.New()
	ta.Placeholder = "Edit your comment..."
	ta.SetValue(render.HNToPlainText(currentText))
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(10)

	return Model{
		textarea: ta,
		itemID:   itemID,
		session:  session,
	}
}

// SetSize sets the viewport dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	tw := w - 4
	if tw > 100 {
		tw = 100
	}
	m.textarea.SetWidth(tw)
	th := h - 8
	if th < 5 {
		th = 5
	}
	m.textarea.SetHeight(th)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				m.err = "Comment cannot be empty"
				return m, nil
			}
			if m.submitting {
				return m, nil
			}
			m.submitting = true
			m.err = ""
			session := m.session
			itemID := m.itemID
			return m, func() tea.Msg {
				err := session.Edit(itemID, text)
				return messages.EditResultMsg{ItemID: itemID, Err: err}
			}
		}

	case messages.EditResultMsg:
		m.submitting = false
		if msg.Err != nil {
			m.err = msg.Err.Error()
			log.Printf("edit error (item=%d): %v", m.itemID, msg.Err)
			return m, nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the edit form.
func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Edit Comment"))
	sb.WriteString("\n\n")
	sb.WriteString(m.textarea.View())
	sb.WriteString("\n\n")

	if m.err != "" {
		sb.WriteString(errorStyle.Render(m.err))
		sb.WriteString("\n")
	}

	if m.submitting {
		sb.WriteString("Submitting...")
	} else {
		sb.WriteString(hintStyle.Render("Ctrl+S to save | Esc to cancel"))
	}

	content := sb.String()
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
