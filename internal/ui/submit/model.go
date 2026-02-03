package submit

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fragmede/nitpick/internal/auth"
	"github.com/fragmede/nitpick/internal/ui/messages"
)

var (
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Width(8)
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#828282"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
)

type field int

const (
	fieldTitle field = iota
	fieldURL
	fieldText
)

// Model is the submit story form.
type Model struct {
	titleInput textinput.Model
	urlInput   textinput.Model
	textInput  textinput.Model
	focused    field
	session    *auth.Session
	err        string
	submitting bool
	width      int
	height     int
}

// New creates a new submit form.
func New(session *auth.Session) Model {
	ti := textinput.New()
	ti.Placeholder = "Story title"
	ti.Focus()
	ti.CharLimit = 80
	ti.Width = 60

	ui := textinput.New()
	ui.Placeholder = "https://example.com (leave blank for text post)"
	ui.CharLimit = 512
	ui.Width = 60

	tx := textinput.New()
	tx.Placeholder = "Text (for Ask HN, optional if URL provided)"
	tx.CharLimit = 2000
	tx.Width = 60

	return Model{
		titleInput: ti,
		urlInput:   ui,
		textInput:  tx,
		focused:    fieldTitle,
		session:    session,
	}
}

// SetSize sets the viewport dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	fw := w - 14
	if fw > 80 {
		fw = 80
	}
	m.titleInput.Width = fw
	m.urlInput.Width = fw
	m.textInput.Width = fw
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focused = (m.focused + 1) % 3
			return m, m.updateFocus()
		case "shift+tab":
			m.focused = (m.focused + 2) % 3
			return m, m.updateFocus()
		case "ctrl+s":
			title := strings.TrimSpace(m.titleInput.Value())
			if title == "" {
				m.err = "Title is required"
				return m, nil
			}
			urlVal := strings.TrimSpace(m.urlInput.Value())
			textVal := strings.TrimSpace(m.textInput.Value())
			if urlVal == "" && textVal == "" {
				m.err = "URL or text is required"
				return m, nil
			}
			if m.submitting {
				return m, nil
			}
			m.submitting = true
			m.err = ""
			session := m.session
			return m, func() tea.Msg {
				err := session.Submit(title, urlVal, textVal)
				return messages.SubmitResultMsg{Err: err}
			}
		}

	case messages.SubmitResultMsg:
		m.submitting = false
		if msg.Err != nil {
			m.err = msg.Err.Error()
			return m, nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	switch m.focused {
	case fieldTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case fieldURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case fieldText:
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) updateFocus() tea.Cmd {
	m.titleInput.Blur()
	m.urlInput.Blur()
	m.textInput.Blur()
	switch m.focused {
	case fieldTitle:
		return m.titleInput.Focus()
	case fieldURL:
		return m.urlInput.Focus()
	case fieldText:
		return m.textInput.Focus()
	}
	return nil
}

// View renders the submit form.
func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Submit Story"))
	sb.WriteString("\n\n")

	sb.WriteString(labelStyle.Render("title") + " " + m.titleInput.View())
	sb.WriteString("\n\n")

	sb.WriteString(labelStyle.Render("url") + " " + m.urlInput.View())
	sb.WriteString("\n\n")

	sb.WriteString(labelStyle.Render("text") + " " + m.textInput.View())
	sb.WriteString("\n\n")

	if m.err != "" {
		sb.WriteString(errorStyle.Render(m.err))
		sb.WriteString("\n")
	}

	if m.submitting {
		sb.WriteString("Submitting...")
	} else {
		sb.WriteString(hintStyle.Render("Tab to switch fields | Ctrl+S to submit | Esc to cancel"))
	}

	content := sb.String()
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
