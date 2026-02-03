package ui

import (
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fragmede/hn-tui/internal/api"
	"github.com/fragmede/hn-tui/internal/auth"
	"github.com/fragmede/hn-tui/internal/cache"
	"github.com/fragmede/hn-tui/internal/config"
	"github.com/fragmede/hn-tui/internal/monitor"
	"github.com/fragmede/hn-tui/internal/ui/login"
	"github.com/fragmede/hn-tui/internal/ui/messages"
	"github.com/fragmede/hn-tui/internal/ui/notifications"
	"github.com/fragmede/hn-tui/internal/ui/reply"
	"github.com/fragmede/hn-tui/internal/ui/statusbar"
	"github.com/fragmede/hn-tui/internal/ui/storylist"
	"github.com/fragmede/hn-tui/internal/ui/storyview"
	"github.com/fragmede/hn-tui/internal/ui/userprofile"
)

// ViewType identifies the active view.
type ViewType int

const (
	ViewStoryList ViewType = iota
	ViewStoryDetail
	ViewLogin
	ViewReply
	ViewNotifications
	ViewUserProfile
)

// App is the root Bubble Tea model.
type App struct {
	// View state
	activeView    ViewType
	previousViews []ViewType

	// Child models
	storyList     storylist.Model
	storyView     storyview.Model
	loginForm     login.Model
	replyForm     reply.Model
	notifications notifications.Model
	userProfile   userprofile.Model
	statusBar     statusbar.Model

	// Shared state
	cfg         config.Config
	client      *api.Client
	cache       *cache.DB
	session     *auth.Session
	monitor     *monitor.Monitor
	unreadCount int

	// Dimensions
	width  int
	height int

	// For passing program reference to monitor
	program *tea.Program
}

// NewApp creates the root application model.
func NewApp(cfg config.Config, client *api.Client, db *cache.DB) *App {
	session := auth.NewSession()
	mon := monitor.New(cfg, client, db)

	return &App{
		activeView:    ViewStoryList,
		storyList:     storylist.New(cfg, client, db),
		statusBar:     statusbar.New(),
		notifications: notifications.New(db),
		cfg:           cfg,
		client:        client,
		cache:         db,
		session:       session,
		monitor:       mon,
	}
}

// SetProgram stores the tea.Program reference for the background monitor.
func (a *App) SetProgram(p *tea.Program) {
	a.program = p
}

// Init starts the application.
func (a *App) Init() tea.Cmd {
	return tea.Batch(a.storyList.Init(), a.tryRestoreSession())
}

func (a *App) tryRestoreSession() tea.Cmd {
	session := a.session
	path := a.cfg.SessionPath
	return func() tea.Msg {
		if session.Load(path) {
			return messages.SessionRestoredMsg{Username: session.Username}
		}
		return nil
	}
}

// Update handles all messages.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		contentHeight := msg.Height - 1 // Reserve 1 line for status bar.
		// Always resize the story list and status bar.
		a.storyList.SetSize(msg.Width, contentHeight)
		a.statusBar.SetSize(msg.Width)
		// Only resize lazily-created views if they're currently active.
		switch a.activeView {
		case ViewStoryDetail:
			a.storyView.SetSize(msg.Width, contentHeight)
		case ViewLogin:
			a.loginForm.SetSize(msg.Width, contentHeight)
		case ViewReply:
			a.replyForm.SetSize(msg.Width, contentHeight)
		case ViewNotifications:
			a.notifications.SetSize(msg.Width, contentHeight)
		case ViewUserProfile:
			a.userProfile.SetSize(msg.Width, contentHeight)
		}
		return a, nil

	case tea.KeyMsg:
		// Global keys (only when not in text input views).
		if a.activeView != ViewLogin && a.activeView != ViewReply {
			switch msg.String() {
			case "ctrl+c":
				a.monitor.Stop()
				return a, tea.Quit
			case "q":
				if a.activeView == ViewStoryList {
					a.monitor.Stop()
					return a, tea.Quit
				}
				return a, a.goBack()
			case "esc":
				if len(a.previousViews) > 0 {
					return a, a.goBack()
				}
				if a.activeView != ViewStoryList {
					a.activeView = ViewStoryList
					return a, nil
				}
			case "?":
				// TODO: help overlay
				return a, nil
			case "tab":
				return a, a.nextTab()
			case "shift+tab":
				return a, a.prevTab()
			case "1":
				return a, a.switchTab(api.StoryTypeTop)
			case "2":
				return a, a.switchTab(api.StoryTypeNew)
			case "3":
				return a, a.switchTab(api.StoryTypeBest)
			case "4":
				return a, a.switchTab(api.StoryTypeAsk)
			case "5":
				return a, a.switchTab(api.StoryTypeShow)
			case "6":
				return a, a.switchTab(api.StoryTypeJobs)
			case "L":
				if !a.session.LoggedIn {
					a.pushView(ViewLogin)
					a.loginForm = login.New(a.session)
					a.loginForm.SetSize(a.width, a.height-1)
				}
				return a, nil
			case "n":
				a.pushView(ViewNotifications)
				a.notifications.Load()
				return a, nil
			}
		} else {
			// Esc in text input views goes back.
			if msg.String() == "esc" {
				return a, a.goBack()
			}
			if msg.String() == "ctrl+c" {
				a.monitor.Stop()
				return a, tea.Quit
			}
		}

	// View transitions.
	case messages.OpenStoryMsg:
		a.pushView(ViewStoryDetail)
		a.storyView = storyview.New(msg.StoryID, a.cfg, a.client, a.cache)
		a.storyView.SetSize(a.width, a.height-1)
		cmd := a.storyView.Init(msg.StoryID)
		return a, cmd

	case messages.GoBackMsg:
		return a, a.goBack()

	case messages.OpenReplyMsg:
		if !a.session.LoggedIn {
			a.pushView(ViewLogin)
			a.loginForm = login.New(a.session)
			a.loginForm.SetSize(a.width, a.height-1)
			return a, nil
		}
		a.pushView(ViewReply)
		a.replyForm = reply.New(msg.ParentID, a.session)
		a.replyForm.SetSize(a.width, a.height-1)
		return a, nil

	case messages.OpenUserMsg:
		a.pushView(ViewUserProfile)
		a.userProfile = userprofile.New(msg.Username, a.cfg, a.client, a.cache)
		a.userProfile.SetSize(a.width, a.height-1)
		cmd := a.userProfile.Init()
		return a, cmd

	case messages.SessionRestoredMsg:
		a.statusBar.SetUser(msg.Username)
		if a.program != nil {
			a.monitor.Start(a.program, msg.Username)
			go a.monitor.SeedComments()
		}
		return a, nil

	case messages.LoginResultMsg:
		if msg.Err == nil {
			a.statusBar.SetUser(msg.Username)
			a.session.Save(a.cfg.SessionPath)
			// Start monitoring.
			if a.program != nil {
				a.monitor.Start(a.program, msg.Username)
				go a.monitor.SeedComments()
			}
			return a, a.goBack()
		}
		// Let login form handle the error.

	case messages.ReplyResultMsg:
		if msg.Err == nil {
			// Track the new comment for monitoring.
			return a, a.goBack()
		}

	case messages.VoteResultMsg:
		if msg.Err != nil {
			a.statusBar.SetStatus("Vote failed: " + msg.Err.Error())
		} else {
			a.statusBar.SetStatus("Voted!")
		}

	case messages.NewNotificationMsg:
		a.unreadCount = msg.UnreadCount
		a.statusBar.SetUnread(msg.UnreadCount)

	case messages.StatusMsg:
		a.statusBar.SetStatus(msg.Text)
		if msg.Text != "" && !msg.IsError {
			// Try to open URL.
			if len(msg.Text) > 9 && msg.Text[:9] == "Opening: " {
				url := msg.Text[9:]
				go openBrowser(url)
			}
		}
	}

	// Route to active view.
	var cmd tea.Cmd
	switch a.activeView {
	case ViewStoryList:
		a.storyList, cmd = a.storyList.Update(msg)
		cmds = append(cmds, cmd)
		a.statusBar.SetActiveTab(a.storyList.StoryType())
	case ViewStoryDetail:
		a.storyView, cmd = a.storyView.Update(msg)
		cmds = append(cmds, cmd)
	case ViewLogin:
		a.loginForm, cmd = a.loginForm.Update(msg)
		cmds = append(cmds, cmd)
	case ViewReply:
		a.replyForm, cmd = a.replyForm.Update(msg)
		cmds = append(cmds, cmd)
	case ViewNotifications:
		a.notifications, cmd = a.notifications.Update(msg)
		cmds = append(cmds, cmd)
	case ViewUserProfile:
		a.userProfile, cmd = a.userProfile.Update(msg)
		cmds = append(cmds, cmd)
	}

	a.statusBar, cmd = a.statusBar.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

// View renders the application.
func (a *App) View() string {
	var content string
	switch a.activeView {
	case ViewStoryList:
		content = a.storyList.View()
	case ViewStoryDetail:
		content = a.storyView.View()
	case ViewLogin:
		content = a.loginForm.View()
	case ViewReply:
		content = a.replyForm.View()
	case ViewNotifications:
		content = a.notifications.View()
	case ViewUserProfile:
		content = a.userProfile.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, a.statusBar.View())
}

func (a *App) pushView(v ViewType) {
	a.previousViews = append(a.previousViews, a.activeView)
	a.activeView = v
}

func (a *App) goBack() tea.Cmd {
	if len(a.previousViews) > 0 {
		a.activeView = a.previousViews[len(a.previousViews)-1]
		a.previousViews = a.previousViews[:len(a.previousViews)-1]
	}
	return nil
}

var tabOrder = []api.StoryType{
	api.StoryTypeTop, api.StoryTypeNew, api.StoryTypeBest,
	api.StoryTypeAsk, api.StoryTypeShow, api.StoryTypeJobs,
}

func (a *App) nextTab() tea.Cmd {
	current := a.storyList.StoryType()
	for i, st := range tabOrder {
		if st == current {
			next := tabOrder[(i+1)%len(tabOrder)]
			return a.switchTab(next)
		}
	}
	return a.switchTab(tabOrder[0])
}

func (a *App) prevTab() tea.Cmd {
	current := a.storyList.StoryType()
	for i, st := range tabOrder {
		if st == current {
			prev := tabOrder[(i-1+len(tabOrder))%len(tabOrder)]
			return a.switchTab(prev)
		}
	}
	return a.switchTab(tabOrder[0])
}

func (a *App) switchTab(st api.StoryType) tea.Cmd {
	if a.activeView != ViewStoryList {
		a.activeView = ViewStoryList
		a.previousViews = nil
	}
	m, cmd := a.storyList.Update(messages.SwitchTabMsg{StoryType: st})
	a.storyList = m
	a.statusBar.SetActiveTab(st)
	return cmd
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	cmd.Run()
}
