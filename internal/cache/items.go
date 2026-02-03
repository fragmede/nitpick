package cache

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/fragmede/hn-tui/internal/api"
)

// GetItem retrieves a cached item. Returns (item, isFresh, error).
// isFresh indicates whether the item is within its TTL.
// Returns nil item on cache miss.
func (d *DB) GetItem(id int, ttl time.Duration) (*api.Item, bool, error) {
	row := d.db.QueryRow(`SELECT id, type, by_user, time_unix, text, parent_id, url,
		title, score, descendants, kids, dead, deleted, fetched_at
		FROM items WHERE id = ?`, id)

	var item api.Item
	var byUser, text, url, title, kids sql.NullString
	var parentID sql.NullInt64
	var fetchedAt int64
	var dead, deleted int

	err := row.Scan(&item.ID, &item.Type, &byUser, &item.Time, &text, &parentID,
		&url, &title, &item.Score, &item.Descendants, &kids, &dead, &deleted, &fetchedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	item.By = byUser.String
	item.Text = text.String
	item.URL = url.String
	item.Title = title.String
	item.Dead = dead != 0
	item.Deleted = deleted != 0
	if parentID.Valid {
		item.Parent = int(parentID.Int64)
	}
	if kids.Valid && kids.String != "" {
		item.RawKids = json.RawMessage(kids.String)
	}

	isFresh := time.Since(time.Unix(fetchedAt, 0)) < ttl
	return &item, isFresh, nil
}

// PutItem stores an item in the cache.
func (d *DB) PutItem(item *api.Item) error {
	now := time.Now().Unix()
	var dead, deleted int
	if item.Dead {
		dead = 1
	}
	if item.Deleted {
		deleted = 1
	}
	kidsJSON := item.KidsJSON()

	_, err := d.db.Exec(`INSERT OR REPLACE INTO items
		(id, type, by_user, time_unix, text, parent_id, url, title, score, descendants, kids, dead, deleted, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Type, nullStr(item.By), item.Time, nullStr(item.Text),
		nullInt(item.Parent), nullStr(item.URL), nullStr(item.Title),
		item.Score, item.Descendants, kidsJSON, dead, deleted, now)
	return err
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}
