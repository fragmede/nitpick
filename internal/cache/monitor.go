package cache

import (
	"encoding/json"
	"time"
)

// MonitoredComment represents a comment being tracked for replies.
type MonitoredComment struct {
	ItemID        int
	ParentStoryID int
	KnownKids     []int
	LastChecked   time.Time
	Depth         int
	CreatedAt     time.Time
}

// GetMonitoredComments returns comments due for checking, ordered by oldest check first.
func (d *DB) GetMonitoredComments(limit int) ([]MonitoredComment, error) {
	rows, err := d.db.Query(`SELECT item_id, parent_story_id, known_kids, last_checked, depth, created_at
		FROM monitored_comments ORDER BY last_checked ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MonitoredComment
	for rows.Next() {
		var mc MonitoredComment
		var kidsJSON string
		var lastChecked, createdAt int64
		if err := rows.Scan(&mc.ItemID, &mc.ParentStoryID, &kidsJSON, &lastChecked, &mc.Depth, &createdAt); err != nil {
			continue
		}
		json.Unmarshal([]byte(kidsJSON), &mc.KnownKids)
		mc.LastChecked = time.Unix(lastChecked, 0)
		mc.CreatedAt = time.Unix(createdAt, 0)
		result = append(result, mc)
	}
	return result, nil
}

// UpsertMonitoredComment inserts or updates a monitored comment.
func (d *DB) UpsertMonitoredComment(mc MonitoredComment) error {
	kidsJSON, _ := json.Marshal(mc.KnownKids)
	_, err := d.db.Exec(`INSERT OR REPLACE INTO monitored_comments
		(item_id, parent_story_id, known_kids, last_checked, depth, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		mc.ItemID, mc.ParentStoryID, string(kidsJSON),
		mc.LastChecked.Unix(), mc.Depth, mc.CreatedAt.Unix())
	return err
}

// AddNotification inserts a new notification.
func (d *DB) AddNotification(itemID, parentID, storyID int, byUser, textPreview string, createdAt int64) error {
	_, err := d.db.Exec(`INSERT OR IGNORE INTO notifications
		(item_id, parent_id, story_id, by_user, text_preview, created_at, read)
		VALUES (?, ?, ?, ?, ?, ?, 0)`,
		itemID, parentID, storyID, byUser, textPreview, createdAt)
	return err
}

// UnreadNotificationCount returns the count of unread notifications.
func (d *DB) UnreadNotificationCount() int {
	var count int
	d.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE read = 0`).Scan(&count)
	return count
}
