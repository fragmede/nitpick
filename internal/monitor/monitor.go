package monitor

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/fragmede/nitpick/internal/api"
	"github.com/fragmede/nitpick/internal/cache"
	"github.com/fragmede/nitpick/internal/config"
	"github.com/fragmede/nitpick/internal/render"
	"github.com/fragmede/nitpick/internal/ui/messages"
)

// Monitor polls for new replies to the user's comments.
type Monitor struct {
	client   *api.Client
	cache    *cache.DB
	cfg      config.Config
	program  *tea.Program
	username string
	stopCh   chan struct{}
}

// New creates a new background monitor.
func New(cfg config.Config, client *api.Client, db *cache.DB) *Monitor {
	return &Monitor{
		client: client,
		cache:  db,
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

// Start begins the background polling loop.
func (m *Monitor) Start(program *tea.Program, username string) {
	m.program = program
	m.username = username
	go m.loop()
}

// Stop halts the background polling.
func (m *Monitor) Stop() {
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
}

// SeedComments adds the user's recent comments to the monitoring list.
func (m *Monitor) SeedComments() {
	ctx := context.Background()
	user, err := m.client.GetUser(ctx, m.username)
	if err != nil {
		return
	}

	// Take the most recent N submitted items.
	limit := m.cfg.MonitorSeedCount
	submitted := user.Submitted
	if len(submitted) > limit {
		submitted = submitted[:limit]
	}

	// Fetch items and filter for comments.
	items, _ := m.client.BatchGetItems(ctx, submitted)
	now := time.Now()
	for _, item := range items {
		if item == nil || item.Type != "comment" {
			continue
		}
		mc := cache.MonitoredComment{
			ItemID:        item.ID,
			ParentStoryID: findStoryID(item, m.cache, m.cfg),
			KnownKids:     item.Kids(),
			LastChecked:   now,
			Depth:         0,
			CreatedAt:     now,
		}
		m.cache.UpsertMonitoredComment(mc)
	}
}

func (m *Monitor) loop() {
	ticker := time.NewTicker(m.cfg.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.poll()
		}
	}
}

func (m *Monitor) poll() {
	comments, err := m.cache.GetMonitoredComments(20)
	if err != nil || len(comments) == 0 {
		return
	}

	ctx := context.Background()
	for _, mc := range comments {
		select {
		case <-m.stopCh:
			return
		default:
		}

		item, err := m.client.GetItem(ctx, mc.ItemID)
		if err != nil {
			continue
		}
		m.cache.PutItem(item)

		// Find new kids.
		knownSet := make(map[int]bool, len(mc.KnownKids))
		for _, kid := range mc.KnownKids {
			knownSet[kid] = true
		}

		currentKids := item.Kids()
		var newKids []int
		for _, kid := range currentKids {
			if !knownSet[kid] {
				newKids = append(newKids, kid)
			}
		}

		if len(newKids) > 0 {
			// Fetch new reply items.
			newItems, _ := m.client.BatchGetItems(ctx, newKids)
			for _, newItem := range newItems {
				if newItem == nil {
					continue
				}
				m.cache.PutItem(newItem)

				// Create notification.
				preview := render.HNToText(newItem.Text, 200)
				if len(preview) > 200 {
					preview = preview[:200]
				}
				m.cache.AddNotification(
					newItem.ID, mc.ItemID, mc.ParentStoryID,
					newItem.By, preview, newItem.Time,
				)

				// Track replies-to-replies if within depth limit.
				if mc.Depth < m.cfg.MonitorMaxDepth {
					newMC := cache.MonitoredComment{
						ItemID:        newItem.ID,
						ParentStoryID: mc.ParentStoryID,
						KnownKids:     newItem.Kids(),
						LastChecked:   time.Now(),
						Depth:         mc.Depth + 1,
						CreatedAt:     time.Now(),
					}
					m.cache.UpsertMonitoredComment(newMC)
				}
			}

			// Notify the TUI.
			if m.program != nil {
				unread := m.cache.UnreadNotificationCount()
				m.program.Send(messages.NewNotificationMsg{UnreadCount: unread})
			}
		}

		// Update the monitored comment.
		mc.KnownKids = currentKids
		mc.LastChecked = time.Now()
		m.cache.UpsertMonitoredComment(mc)
	}
}

// findStoryID walks up the parent chain to find the root story ID.
func findStoryID(item *api.Item, db *cache.DB, cfg config.Config) int {
	current := item
	for current.Parent != 0 {
		parent, _, _ := db.GetItem(current.Parent, cfg.ItemTTL)
		if parent == nil {
			return current.Parent
		}
		if parent.Type == "story" {
			return parent.ID
		}
		current = parent
	}
	return current.ID
}
