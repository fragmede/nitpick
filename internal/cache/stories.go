package cache

import (
	"database/sql"
	"encoding/json"
	"time"
)

// GetStoryList retrieves cached story IDs for a list type.
// Returns (ids, isFresh, error). ids is nil on cache miss.
func (d *DB) GetStoryList(listType string, ttl time.Duration) ([]int, bool, error) {
	row := d.db.QueryRow(`SELECT item_ids, fetched_at FROM story_lists WHERE list_type = ?`, listType)

	var idsJSON string
	var fetchedAt int64
	err := row.Scan(&idsJSON, &fetchedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var ids []int
	if err := json.Unmarshal([]byte(idsJSON), &ids); err != nil {
		return nil, false, err
	}

	isFresh := time.Since(time.Unix(fetchedAt, 0)) < ttl
	return ids, isFresh, nil
}

// PutStoryList stores a story ID list in the cache.
func (d *DB) PutStoryList(listType string, ids []int) error {
	idsJSON, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`INSERT OR REPLACE INTO story_lists (list_type, item_ids, fetched_at) VALUES (?, ?, ?)`,
		listType, string(idsJSON), time.Now().Unix())
	return err
}
