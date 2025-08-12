package textutil

import (
	"regexp"
	"strings"
)

var (
	reHTMLTags   = regexp.MustCompile(`<[^>]+>`) // simple stripper
	reWhitespace = regexp.MustCompile(`\s+`)
	// Emoji ranges (basic) — extend if needed
	reEmoji = regexp.MustCompile("[\u2190-\u21FF\u2600-\u27BF\u1F300-\u1F6FF\u1F900-\u1F9FF\u1FA70-\u1FAFF]")
)

func Clean(s string, stripHTML, stripEmoji, normWS bool, maxLen, minLen int) (out string, ok bool) {
	out = s
	if stripHTML {
		out = reHTMLTags.ReplaceAllString(out, " ")
	}
	if stripEmoji {
		out = reEmoji.ReplaceAllString(out, " ")
	}
	if normWS {
		out = reWhitespace.ReplaceAllString(strings.TrimSpace(out), " ")
	}
	if maxLen > 0 && len(out) > maxLen {
		out = out[:maxLen]
	}
	if len(out) < minLen {
		return out, false
	}
	return out, true
}
