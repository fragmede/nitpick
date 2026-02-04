package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const hnBaseURL = "https://news.ycombinator.com"

// ThreadComment represents a parsed comment from the HN threads page.
type ThreadComment struct {
	ID         int
	Indent     int
	Author     string
	Time       int64
	Score      int
	Text       string // raw HN HTML
	StoryTitle string
	StoryID    int
}

var (
	indentRe   = regexp.MustCompile(`class="ind" indent="(\d+)"`)
	hnuserRe   = regexp.MustCompile(`class="hnuser">([^<]+)</a>`)
	ageRe      = regexp.MustCompile(`class="age" title="[^ ]+ (\d+)"`)
	onstoryRe  = regexp.MustCompile(`class="onstory">[^<]*on:\s*<a href="item\?id=(\d+)"[^>]*title="([^"]*)"`)
	scoreRe    = regexp.MustCompile(`class="score"[^>]*>(\d+)\s+point`)
	commtextRe = regexp.MustCompile(`(?s)class="commtext[^"]*">(.*?)</div>\s*<div class="reply">`)
)

// GetThreadsPage scrapes the HN threads page for a user and returns
// comments with their proper indent levels as shown on the site.
func (c *Client) GetThreadsPage(ctx context.Context, username string) ([]ThreadComment, error) {
	url := fmt.Sprintf("%s/threads?id=%s", hnBaseURL, username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "nitpick/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching threads page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("threads page returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading threads page: %w", err)
	}

	return ParseThreadsHTML(string(body))
}

// ParseThreadsHTML extracts comments from the HN threads page HTML.
func ParseThreadsHTML(body string) ([]ThreadComment, error) {
	// Split by comment rows. Each comment starts with class="athing comtr".
	parts := strings.Split(body, `class="athing comtr" `)
	if len(parts) < 2 {
		return nil, fmt.Errorf("no comments found on threads page")
	}

	var comments []ThreadComment
	for _, part := range parts[1:] {
		tc := ThreadComment{}

		// Comment ID: id="NNNN" at the start of the chunk.
		if idx := strings.Index(part, `id="`); idx >= 0 {
			end := strings.Index(part[idx+4:], `"`)
			if end > 0 {
				tc.ID, _ = strconv.Atoi(part[idx+4 : idx+4+end])
			}
		}
		if tc.ID == 0 {
			continue
		}

		// Indent level.
		if m := indentRe.FindStringSubmatch(part); len(m) > 1 {
			tc.Indent, _ = strconv.Atoi(m[1])
		}

		// Author.
		if m := hnuserRe.FindStringSubmatch(part); len(m) > 1 {
			tc.Author = m[1]
		}

		// Unix timestamp from age title attribute.
		if m := ageRe.FindStringSubmatch(part); len(m) > 1 {
			tc.Time, _ = strconv.ParseInt(m[1], 10, 64)
		}

		// Score (only shown for the user's own comments).
		if m := scoreRe.FindStringSubmatch(part); len(m) > 1 {
			tc.Score, _ = strconv.Atoi(m[1])
		}

		// Story title and ID (only on user's own comments, not context replies).
		if m := onstoryRe.FindStringSubmatch(part); len(m) > 2 {
			tc.StoryID, _ = strconv.Atoi(m[1])
			tc.StoryTitle = m[2]
		}

		// Comment text (raw HTML).
		if m := commtextRe.FindStringSubmatch(part); len(m) > 1 {
			tc.Text = m[1]
		}

		comments = append(comments, tc)
	}

	return comments, nil
}
