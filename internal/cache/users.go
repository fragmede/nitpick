package cache

import (
	"database/sql"
	"time"

	"github.com/fragmede/hn-tui/internal/api"
)

// GetUser retrieves a cached user profile.
func (d *DB) GetUser(username string, ttl time.Duration) (*api.User, bool, error) {
	row := d.db.QueryRow(`SELECT id, created, karma, about, fetched_at FROM users WHERE id = ?`, username)

	var user api.User
	var about sql.NullString
	var fetchedAt int64

	err := row.Scan(&user.ID, &user.Created, &user.Karma, &about, &fetchedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	user.About = about.String
	isFresh := time.Since(time.Unix(fetchedAt, 0)) < ttl
	return &user, isFresh, nil
}

// PutUser stores a user profile in the cache.
func (d *DB) PutUser(user *api.User) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO users (id, created, karma, about, fetched_at) VALUES (?, ?, ?, ?, ?)`,
		user.ID, user.Created, user.Karma, nullStr(user.About), time.Now().Unix())
	return err
}
