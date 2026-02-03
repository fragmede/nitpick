package auth

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

const hnBaseURL = "https://news.ycombinator.com"

// Session manages HN authentication state.
type Session struct {
	client   *http.Client
	jar      *cookiejar.Jar
	Username string
	LoggedIn bool
}

// NewSession creates a new auth session.
func NewSession() *Session {
	jar, _ := cookiejar.New(nil)
	return &Session{
		client: &http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		jar: jar,
	}
}

// Login authenticates with HN using username and password.
func (s *Session) Login(username, password string) error {
	data := url.Values{
		"acct": {username},
		"pw":   {password},
		"goto": {"news"},
	}

	resp, err := s.client.PostForm(hnBaseURL+"/login", data)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	// Validate: fetch the main page and check for logout link.
	if err := s.validate(); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	s.Username = username
	s.LoggedIn = true
	return nil
}

func (s *Session) validate() error {
	resp, err := s.client.Get(hnBaseURL + "/news")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if !strings.Contains(string(body), "logout") {
		return fmt.Errorf("authentication failed - no logout link found")
	}
	return nil
}

// Reply posts a reply to an HN item.
func (s *Session) Reply(parentID int, text string) error {
	if !s.LoggedIn {
		return fmt.Errorf("not logged in")
	}

	// First fetch the reply page to get the fnid token.
	replyURL := fmt.Sprintf("%s/reply?id=%d", hnBaseURL, parentID)
	resp, err := s.client.Get(replyURL)
	if err != nil {
		return fmt.Errorf("fetching reply page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fnid := extractFnid(string(body))
	if fnid == "" {
		return fmt.Errorf("could not extract reply token (fnid)")
	}

	// Submit the reply.
	data := url.Values{
		"fnid": {fnid},
		"text": {text},
	}
	resp2, err := s.client.PostForm(hnBaseURL+"/comment", data)
	if err != nil {
		return fmt.Errorf("submitting reply: %w", err)
	}
	defer resp2.Body.Close()
	io.ReadAll(resp2.Body)

	// HN redirects on success.
	if resp2.StatusCode >= 400 {
		return fmt.Errorf("reply failed with status %d", resp2.StatusCode)
	}
	return nil
}

// Vote upvotes an HN item.
func (s *Session) Vote(itemID int) error {
	if !s.LoggedIn {
		return fmt.Errorf("not logged in")
	}

	// Fetch the item page to get the vote auth token.
	itemURL := fmt.Sprintf("%s/item?id=%d", hnBaseURL, itemID)
	resp, err := s.client.Get(itemURL)
	if err != nil {
		return fmt.Errorf("fetching item page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	voteURL := extractVoteURL(string(body), itemID)
	if voteURL == "" {
		return fmt.Errorf("could not find vote link for item %d", itemID)
	}

	// Execute the vote.
	resp2, err := s.client.Get(hnBaseURL + "/" + voteURL)
	if err != nil {
		return fmt.Errorf("voting: %w", err)
	}
	defer resp2.Body.Close()
	io.ReadAll(resp2.Body)

	return nil
}

// GetClient returns the underlying HTTP client (with cookies).
func (s *Session) GetClient() *http.Client {
	return s.client
}

func extractFnid(html string) string {
	// Look for: <input type="hidden" name="fnid" value="XXXX">
	idx := strings.Index(html, `name="fnid"`)
	if idx == -1 {
		return ""
	}
	// Search for value= near this position.
	sub := html[idx:]
	valIdx := strings.Index(sub, `value="`)
	if valIdx == -1 {
		return ""
	}
	start := valIdx + len(`value="`)
	end := strings.Index(sub[start:], `"`)
	if end == -1 {
		return ""
	}
	return sub[start : start+end]
}

func extractVoteURL(html string, itemID int) string {
	// Look for: id='up_ITEMID' ... href='vote?...'
	needle := fmt.Sprintf(`id='up_%d'`, itemID)
	idx := strings.Index(html, needle)
	if idx == -1 {
		return ""
	}
	// Search backwards for href=
	sub := html[:idx]
	hrefIdx := strings.LastIndex(sub, `href='`)
	if hrefIdx == -1 {
		// Try with double quotes.
		hrefIdx = strings.LastIndex(sub, `href="`)
	}
	if hrefIdx == -1 {
		return ""
	}
	start := hrefIdx + 6
	quote := sub[hrefIdx+5]
	endSub := html[start:]
	end := strings.IndexByte(endSub, quote)
	if end == -1 {
		return ""
	}
	return endSub[:end]
}
