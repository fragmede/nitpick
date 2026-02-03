package config

import (
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	CacheDir         string
	DBPath           string
	SessionPath      string
	LogPath          string
	StoryListTTL     time.Duration
	ItemTTL          time.Duration
	CommentTTL       time.Duration
	UserTTL          time.Duration
	MonitorInterval  time.Duration
	MonitorMaxDepth  int
	MonitorSeedCount int
	FetchPageSize    int
}

func Default() Config {
	cacheDir := filepath.Join(userConfigDir(), "nitpick")
	return Config{
		CacheDir:         cacheDir,
		DBPath:           filepath.Join(cacheDir, "cache.db"),
		SessionPath:      filepath.Join(cacheDir, "session.json"),
		LogPath:          filepath.Join(cacheDir, "debug.log"),
		StoryListTTL:     60 * time.Second,
		ItemTTL:          5 * time.Minute,
		CommentTTL:       10 * time.Minute,
		UserTTL:          1 * time.Hour,
		MonitorInterval:  30 * time.Second,
		MonitorMaxDepth:  2,
		MonitorSeedCount: 50,
		FetchPageSize:    30,
	}
}

func userConfigDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}
