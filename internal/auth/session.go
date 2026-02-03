package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
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

// savedSession is the JSON structure written to disk.
type savedSession struct {
	Username string         `json:"username"`
	Cookies  []savedCookie  `json:"cookies"`
	SavedAt  time.Time      `json:"saved_at"`
}

type savedCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires"`
	Secure   bool      `json:"secure"`
	HttpOnly bool      `json:"http_only"`
}

// Save persists the session cookies to a file.
func (s *Session) Save(path string) error {
	if !s.LoggedIn {
		return nil
	}

	u, _ := url.Parse(hnBaseURL)
	cookies := s.jar.Cookies(u)

	sc := make([]savedCookie, len(cookies))
	for i, c := range cookies {
		sc[i] = savedCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
		}
	}

	data, err := json.MarshalIndent(savedSession{
		Username: s.Username,
		Cookies:  sc,
		SavedAt:  time.Now(),
	}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// Load restores a session from a file and validates it's still good.
// Returns true if the session was restored successfully.
func (s *Session) Load(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var saved savedSession
	if err := json.Unmarshal(data, &saved); err != nil {
		return false
	}

	if saved.Username == "" || len(saved.Cookies) == 0 {
		return false
	}

	// Restore cookies into the jar.
	u, _ := url.Parse(hnBaseURL)
	cookies := make([]*http.Cookie, len(saved.Cookies))
	for i, sc := range saved.Cookies {
		cookies[i] = &http.Cookie{
			Name:     sc.Name,
			Value:    sc.Value,
			Domain:   sc.Domain,
			Path:     sc.Path,
			Expires:  sc.Expires,
			Secure:   sc.Secure,
			HttpOnly: sc.HttpOnly,
		}
	}
	s.jar.SetCookies(u, cookies)

	// Validate the session is still alive.
	if err := s.validate(); err != nil {
		// Stale session — clear it.
		os.Remove(path)
		return false
	}

	s.Username = saved.Username
	s.LoggedIn = true
	return true
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

	// Fetch the reply page to get the form tokens.
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

	html := string(body)
	data := extractFormInputs(html)
	if data.Get("hmac") == "" {
		log.Printf("reply page HTML (%d bytes): %s", len(body), html)
		return fmt.Errorf("could not extract reply token (hmac) from reply page (status %d, %d bytes)", resp.StatusCode, len(body))
	}
	data.Set("text", text)
	log.Printf("reply form fields: parent=%s goto=%s hmac=%s text_len=%d",
		data.Get("parent"), data.Get("goto"), data.Get("hmac"), len(text))

	resp2, err := s.client.PostForm(hnBaseURL+"/comment", data)
	if err != nil {
		return fmt.Errorf("submitting reply: %w", err)
	}
	defer resp2.Body.Close()

	respBody, _ := io.ReadAll(resp2.Body)
	log.Printf("reply POST response: status=%d url=%s body_len=%d", resp2.StatusCode, resp2.Request.URL, len(respBody))
	return checkHNResponse(resp2.StatusCode, respBody)
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

// Submit posts a new story to HN.
func (s *Session) Submit(title, storyURL, text string) error {
	if !s.LoggedIn {
		return fmt.Errorf("not logged in")
	}

	// Fetch the submit page to get the fnid token.
	resp, err := s.client.Get(hnBaseURL + "/submit")
	if err != nil {
		return fmt.Errorf("fetching submit page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fnid := extractHiddenInput(string(body), "fnid")
	if fnid == "" {
		return fmt.Errorf("could not extract submit token (fnid) from submit page (status %d, %d bytes)", resp.StatusCode, len(body))
	}

	data := url.Values{
		"fnid":  {fnid},
		"fnop":  {"submit-page"},
		"title": {title},
		"url":   {storyURL},
		"text":  {text},
	}
	resp2, err := s.client.PostForm(hnBaseURL+"/r", data)
	if err != nil {
		return fmt.Errorf("submitting story: %w", err)
	}
	defer resp2.Body.Close()
	io.ReadAll(resp2.Body)

	if resp2.StatusCode >= 400 {
		return fmt.Errorf("submit failed with status %d", resp2.StatusCode)
	}
	return nil
}

// GetClient returns the underlying HTTP client (with cookies).
func (s *Session) GetClient() *http.Client {
	return s.client
}

// extractHiddenInput extracts the value of a hidden input field by name.
// Handles: <input type="hidden" name="X" value="Y"> with attributes in any order.
func extractHiddenInput(html, name string) string {
	needle := fmt.Sprintf(`name="%s"`, name)
	idx := strings.Index(html, needle)
	if idx == -1 {
		return ""
	}
	// Search for value= after name=.
	sub := html[idx:]
	valIdx := strings.Index(sub, `value="`)
	if valIdx == -1 {
		// Try before name= (value may precede name in the same tag).
		before := html[max(0, idx-200):idx]
		valIdx = strings.LastIndex(before, `value="`)
		if valIdx == -1 {
			return ""
		}
		start := valIdx + len(`value="`)
		end := strings.Index(before[start:], `"`)
		if end == -1 {
			return ""
		}
		return before[start : start+end]
	}
	start := valIdx + len(`value="`)
	end := strings.Index(sub[start:], `"`)
	if end == -1 {
		return ""
	}
	return sub[start : start+end]
}

// hiddenInputRe matches <input type="hidden" name="X" value="Y"> with
// attributes in any order. It captures name and value groups.
var hiddenInputRe = regexp.MustCompile(
	`<input[^>]*type=["']?hidden["']?[^>]*name=["']([^"']+)["'][^>]*value=["']([^"']+)["'][^>]*/?>` +
		`|` +
		`<input[^>]*value=["']([^"']+)["'][^>]*name=["']([^"']+)["'][^>]*type=["']?hidden["']?[^>]*/?>` +
		`|` +
		`<input[^>]*name=["']([^"']+)["'][^>]*value=["']([^"']+)["'][^>]*type=["']?hidden["']?[^>]*/?>`,
)

// extractFormInputs extracts all hidden input fields from HTML as url.Values.
func extractFormInputs(html string) url.Values {
	vals := url.Values{}
	// Simple approach: find all input type=hidden and extract name/value.
	// HN uses: <input type="hidden" name="X" value="Y">
	matches := hiddenInputRe.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		// Groups depend on which alternation matched.
		if m[1] != "" && m[2] != "" {
			vals.Set(m[1], m[2])
		} else if m[3] != "" && m[4] != "" {
			vals.Set(m[4], m[3]) // value before name
		} else if m[5] != "" && m[6] != "" {
			vals.Set(m[5], m[6]) // name before value, type last
		}
	}
	return vals
}

// checkHNResponse checks the POST response for HN error messages.
func checkHNResponse(statusCode int, body []byte) error {
	if statusCode >= 400 {
		return fmt.Errorf("request failed with status %d", statusCode)
	}
	s := string(body)
	// HN wraps errors in a simple page body. Common patterns:
	for _, errText := range []string{"Unknown.", "Please try again.", "You're submitting too fast."} {
		if strings.Contains(s, errText) {
			return fmt.Errorf("HN error: %s", errText)
		}
	}
	return nil
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
