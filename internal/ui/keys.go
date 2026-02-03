package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit       key.Binding
	Back       key.Binding
	Help       key.Binding
	Enter      key.Binding
	Refresh    key.Binding
	Login      key.Binding
	Notify     key.Binding
	OpenURL    key.Binding
	Upvote     key.Binding
	Reply      key.Binding
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Home       key.Binding
	End        key.Binding
	Collapse   key.Binding
	Parent     key.Binding
	NextSib    key.Binding
	Tab1       key.Binding
	Tab2       key.Binding
	Tab3       key.Binding
	Tab4       key.Binding
	Tab5       key.Binding
	Tab6       key.Binding
	Submit     key.Binding
	Filter     key.Binding
	UserDetail key.Binding
}

var Keys = KeyMap{
	Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Login:      key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "login")),
	Notify:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "notifications")),
	OpenURL:    key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open url")),
	Upvote:     key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "upvote")),
	Reply:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reply")),
	Up:         key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/up", "up")),
	Down:       key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/down", "down")),
	PageUp:     key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("ctrl+u", "page up")),
	PageDown:   key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d", "page down")),
	Home:       key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
	End:        key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
	Collapse:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "collapse")),
	Parent:     key.NewBinding(key.WithKeys("["), key.WithHelp("[", "parent")),
	NextSib:    key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next sibling")),
	Tab1:       key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "top")),
	Tab2:       key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "new")),
	Tab3:       key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "best")),
	Tab4:       key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "ask")),
	Tab5:       key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "show")),
	Tab6:       key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "jobs")),
	Submit:     key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "submit")),
	Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	UserDetail: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "user profile")),
}
