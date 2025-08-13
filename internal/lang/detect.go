package lang

import (
	wlg "github.com/abadojack/whatlanggo"
)

// DetectCode returns ISO-639-1 code and confidence in [0,1].
// If language cannot be reliably detected, returns "und" and 0.
func DetectCode(text string) (code string, conf float64) {
	if len(text) == 0 {
		return "und", 0
	}
	info := wlg.Detect(text)
	if !info.IsReliable() {
		return "und", 0
	}
	iso6391 := info.Lang.Iso6391()
	if iso6391 == "" {
		return "und", info.Confidence
	}
	return iso6391, info.Confidence
}
