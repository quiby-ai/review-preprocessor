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

// IsContentful applies simple heuristics to determine if text is contentful.
// Rules: non-empty, min words or chars, alpha ratio threshold.
func IsContentful(text string, minWords, minChars int, minAlphaRatio float64) bool {
	if len(strings.TrimSpace(text)) == 0 {
		return false
	}
	if minChars > 0 && len(text) < minChars {
		return false
	}
	// word count
	words := reWhitespace.Split(strings.TrimSpace(text), -1)
	if minWords > 0 && len(words) < minWords {
		return false
	}
	// alpha ratio
	if minAlphaRatio > 0 {
		var alpha int
		for _, r := range text {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
				alpha++
			}
		}
		if total := len(text); total > 0 {
			if float64(alpha)/float64(total) < minAlphaRatio {
				return false
			}
		}
	}
	return true
}
