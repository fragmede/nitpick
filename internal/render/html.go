package render

import (
	"html"
	"strings"

	xhtml "golang.org/x/net/html"
)

// HNToText converts HN's limited HTML to plain text with basic formatting.
// HN uses: <p> (paragraph), <a> (links), <i> (italic), <code> (inline code),
// <pre><code> (code blocks), and HTML entities.
func HNToText(raw string, width int) string {
	if raw == "" {
		return ""
	}

	// Pre-unescape HTML entities.
	raw = html.UnescapeString(raw)

	tokenizer := xhtml.NewTokenizer(strings.NewReader(raw))
	var sb strings.Builder
	var inPre, inCode bool
	var anchorURL string

	for {
		tt := tokenizer.Next()
		switch tt {
		case xhtml.ErrorToken:
			return wrapText(strings.TrimSpace(sb.String()), width)

		case xhtml.StartTagToken:
			t := tokenizer.Token()
			switch t.Data {
			case "p":
				if sb.Len() > 0 {
					sb.WriteString("\n\n")
				}
			case "i", "em":
				sb.WriteString("*")
			case "code":
				if !inPre {
					sb.WriteString("`")
				}
				inCode = true
			case "pre":
				inPre = true
				sb.WriteString("\n")
			case "a":
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						anchorURL = attr.Val
					}
				}
			}

		case xhtml.EndTagToken:
			t := tokenizer.Token()
			switch t.Data {
			case "i", "em":
				sb.WriteString("*")
			case "code":
				if !inPre {
					sb.WriteString("`")
				}
				inCode = false
			case "pre":
				inPre = false
				sb.WriteString("\n")
			case "a":
				if anchorURL != "" {
					text := strings.TrimSpace(sb.String())
					// Only append URL if it differs from the link text.
					if !strings.HasSuffix(text, anchorURL) {
						sb.WriteString(" [")
						sb.WriteString(anchorURL)
						sb.WriteString("]")
					}
				}
	
				anchorURL = ""
			}

		case xhtml.TextToken:
			text := tokenizer.Token().Data
			if inPre {
				// Preserve whitespace in pre blocks, indent with 4 spaces.
				lines := strings.Split(text, "\n")
				for i, line := range lines {
					if i > 0 {
						sb.WriteString("\n")
					}
					if line != "" {
						sb.WriteString("    ")
						sb.WriteString(line)
					}
				}
			} else if inCode {
				sb.WriteString(text)
			} else {
				// Collapse whitespace for normal text.
				sb.WriteString(text)
			}
		}
	}
}

// wrapText performs simple word wrapping to the given width.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	var result strings.Builder
	for _, paragraph := range strings.Split(text, "\n") {
		if strings.HasPrefix(paragraph, "    ") {
			// Don't wrap code blocks.
			result.WriteString(paragraph)
			result.WriteString("\n")
			continue
		}
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			result.WriteString("\n")
			continue
		}
		lineLen := 0
		for i, word := range words {
			wlen := len(word)
			if i > 0 && lineLen+1+wlen > width {
				result.WriteString("\n")
				lineLen = 0
			} else if i > 0 {
				result.WriteString(" ")
				lineLen++
			}
			result.WriteString(word)
			lineLen += wlen
		}
		result.WriteString("\n")
	}
	return strings.TrimRight(result.String(), "\n")
}
