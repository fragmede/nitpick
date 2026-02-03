package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/fragmede/nitpick/internal/api"
	"github.com/fragmede/nitpick/internal/cache"
	"github.com/fragmede/nitpick/internal/config"
	"github.com/fragmede/nitpick/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg := config.Default()

	if err := os.MkdirAll(cfg.CacheDir, 0o755); err != nil {
		log.Fatalf("creating cache dir: %v", err)
	}

	db, err := cache.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("opening cache: %v", err)
	}
	defer db.Close()

	client := api.NewClient()

	// Prefetch top stories into cache on startup.
	go prefetch(client, db)

	app := ui.NewApp(cfg, client, db)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	app.SetProgram(p)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func prefetch(client *api.Client, db *cache.DB) {
	ctx := context.Background()
	ids, err := client.GetStoryIDs(ctx, api.StoryTypeTop)
	if err != nil {
		return
	}
	if len(ids) > 30 {
		ids = ids[:30]
	}
	db.PutStoryList(string(api.StoryTypeTop), ids)
	items, _ := client.BatchGetItems(ctx, ids)
	for _, item := range items {
		if item != nil {
			db.PutItem(item)
		}
	}
}
