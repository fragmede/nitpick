package auth

// Scraper utilities for extracting auth tokens from HN HTML.
// The main extraction functions (extractFnid, extractVoteURL) are in session.go
// since they're tightly coupled to the auth flow.
//
// This file is reserved for any additional scraping utilities that
// may be needed (e.g., extracting comment text for display, parsing
// user profile pages, etc.)
