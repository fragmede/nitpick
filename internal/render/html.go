package render

import (
	"html"
	"strings"

	xhtml "golang.org/x/net/html"
)

// HNToPlainText converts HN's limited HTML to plain text without wrapping.
// Useful for pre-filling edit forms with decoded comment text.
func HNToPlainText(raw string) string {
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
			return strings.TrimSpace(sb.String())

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
				// Collapse internal whitespace, but preserve word boundaries
				// at token edges (e.g. space before/after inline elements).
				normalized := strings.Join(strings.Fields(text), " ")
				if normalized == "" {
					// All-whitespace token (space between tags).
					if sb.Len() > 0 && len(text) > 0 && !endsWithSpaceOrNewline(&sb) {
						sb.WriteString(" ")
					}
				} else {
					// Preserve leading space at token boundary.
					if sb.Len() > 0 && isSpaceByte(text[0]) && !endsWithSpaceOrNewline(&sb) {
						sb.WriteString(" ")
					}
					sb.WriteString(normalized)
					// Preserve trailing space for boundary with next element.
					if isSpaceByte(text[len(text)-1]) {
						sb.WriteString(" ")
					}
				}
			}
		}
	}
}

// HNToText converts HN's limited HTML to plain text with word wrapping.
// HN uses: <p> (paragraph), <a> (links), <i> (italic), <code> (inline code),
// <pre><code> (code blocks), and HTML entities.
func HNToText(raw string, width int) string {
	return wrapText(HNToPlainText(raw), width)
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

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func endsWithSpaceOrNewline(sb *strings.Builder) bool {
	s := sb.String()
	if len(s) == 0 {
		return true // treat empty as "no space needed"
	}
	last := s[len(s)-1]
	return last == ' ' || last == '\n'
}
