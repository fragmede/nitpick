package storylist

import (
	"context"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fragmede/nitpick/internal/api"
	"github.com/fragmede/nitpick/internal/cache"
	"github.com/fragmede/nitpick/internal/config"
	"github.com/fragmede/nitpick/internal/ui/messages"
)

// Model is the story list view.
type Model struct {
	list      list.Model
	storyType api.StoryType
	client    *api.Client
	cache     *cache.DB
	cfg       config.Config
	loading   bool
	width     int
	height    int
}

// New creates a new story list model.
func New(cfg config.Config, client *api.Client, db *cache.DB) Model {
	delegate := Delegate{}
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Hacker News"
	l.SetShowStatusBar(true)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	return Model{
		list:      l,
		storyType: api.StoryTypeTop,
		client:    client,
		cache:     db,
		cfg:       cfg,
	}
}

// Init loads the initial story list.
func (m Model) Init() tea.Cmd {
	return m.loadStories()
}

// SetSize updates the viewport dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w, h)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.StoriesLoadedMsg:
		if msg.Err != nil {
			m.list.Title = "Error: " + msg.Err.Error()
			m.loading = false
			return m, nil
		}
		if msg.StoryType != m.storyType {
			return m, nil
		}
		items := make([]list.Item, 0, len(msg.Items))
		for i, item := range msg.Items {
			if item != nil {
				items = append(items, StoryItem{Item: item, Index: i})
			}
		}
		m.list.SetItems(items)
		m.list.Title = storyTypeTitle(m.storyType)
		m.loading = false
		return m, nil

	case messages.SwitchTabMsg:
		m.storyType = msg.StoryType
		m.list.Title = storyTypeTitle(m.storyType) + " (loading...)"
		m.loading = true
		return m, m.loadStories()

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(StoryItem); ok {
				return m, func() tea.Msg {
					return messages.OpenStoryMsg{StoryID: item.Item.ID}
				}
			}
		case "o":
			if item, ok := m.list.SelectedItem().(StoryItem); ok && item.Item.URL != "" {
				return m, func() tea.Msg {
					return messages.StatusMsg{Text: "Opening: " + item.Item.URL}
				}
			}
		case "r", "ctrl+r":
			m.loading = true
			m.list.Title = storyTypeTitle(m.storyType) + " (refreshing...)"
			return m, m.loadStoriesForce()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the story list.
func (m Model) View() string {
	return m.list.View()
}

// StoryType returns the current story type.
func (m Model) StoryType() api.StoryType {
	return m.storyType
}

func (m Model) loadStories() tea.Cmd {
	st := m.storyType
	client := m.client
	db := m.cache
	cfg := m.cfg

	// Past stories use Algolia (returns stories, not comments).
	if st == api.StoryTypePast {
		return func() tea.Msg {
			items, err := client.GetPastStories(context.Background(), cfg.FetchPageSize)
			return messages.StoriesLoadedMsg{StoryType: st, Items: items, Err: err}
		}
	}

	return func() tea.Msg {
		// Standard Firebase API path: try cache first.
		ids, fresh, _ := db.GetStoryList(string(st), cfg.StoryListTTL)
		if fresh && len(ids) > 0 {
			return loadItemsFromCache(st, ids, db, cfg)
		}
		return fetchAndCache(st, client, db, cfg, ids)
	}
}

func (m Model) loadStoriesForce() tea.Cmd {
	st := m.storyType
	client := m.client
	db := m.cache
	cfg := m.cfg

	if st == api.StoryTypePast {
		return func() tea.Msg {
			items, err := client.GetPastStories(context.Background(), cfg.FetchPageSize)
			return messages.StoriesLoadedMsg{StoryType: st, Items: items, Err: err}
		}
	}

	return func() tea.Msg {
		db.InvalidateStoryList(string(st))
		return fetchAndCache(st, client, db, cfg, nil)
	}
}

func loadItemsFromCache(st api.StoryType, ids []int, db *cache.DB, cfg config.Config) messages.StoriesLoadedMsg {
	limit := cfg.FetchPageSize
	if limit > len(ids) {
		limit = len(ids)
	}
	items := make([]*api.Item, limit)
	for i := 0; i < limit; i++ {
		item, _, _ := db.GetItem(ids[i], cfg.ItemTTL)
		items[i] = item
	}
	return messages.StoriesLoadedMsg{StoryType: st, Items: items}
}

func fetchAndCache(st api.StoryType, client *api.Client, db *cache.DB, cfg config.Config, fallbackIDs []int) messages.StoriesLoadedMsg {
	ctx := context.Background()
	ids, err := client.GetStoryIDs(ctx, st)
	if err != nil {
		// Use cached IDs if available.
		if len(fallbackIDs) > 0 {
			return loadItemsFromCache(st, fallbackIDs, db, cfg)
		}
		return messages.StoriesLoadedMsg{StoryType: st, Err: err}
	}

	limit := cfg.FetchPageSize
	if limit > len(ids) {
		limit = len(ids)
	}
	db.PutStoryList(string(st), ids)

	fetchIDs := ids[:limit]
	items, err := client.BatchGetItems(ctx, fetchIDs)
	if err != nil {
		return messages.StoriesLoadedMsg{StoryType: st, Err: err}
	}
	for _, item := range items {
		if item != nil {
			db.PutItem(item)
		}
	}
	return messages.StoriesLoadedMsg{StoryType: st, Items: items}
}

func storyTypeTitle(st api.StoryType) string {
	switch st {
	case api.StoryTypeTop:
		return "Top Stories"
	case api.StoryTypeNew:
		return "New"
	case api.StoryTypeBest:
		return "Best Stories"
	case api.StoryTypeAsk:
		return "Ask HN"
	case api.StoryTypeShow:
		return "Show HN"
	case api.StoryTypeJobs:
		return "Jobs"
	case api.StoryTypePast:
		return "Past"
	default:
		return "Hacker News"
	}
}
