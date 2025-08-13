package translate

import (
	"context"
)

// Cascade calls Primary translator first, and retries a small sampled subset
// with Fallback if adequacy seems poor or Primary failed.
type Cascade struct {
	Primary       Translator
	Fallback      Translator
	Sample        int
	AdequacyRatio float64
}

func NewCascade(primary, fallback Translator, sample int, adequacyRatio float64) *Cascade {
	return &Cascade{Primary: primary, Fallback: fallback, Sample: sample, AdequacyRatio: adequacyRatio}
}

func (c *Cascade) TranslateBatch(ctx context.Context, items []Item, target string) (map[string]Result, error) {
	if c.Primary == nil {
		return map[string]Result{}, nil
	}
	res, err := c.Primary.TranslateBatch(ctx, items, target)
	if err != nil {
		// On request/limit/error – use fallback if configured
		if c.Fallback != nil {
			return c.Fallback.TranslateBatch(ctx, items, target)
		}
		return res, err
	}
	// Evaluate adequacy for a small sample; if inadequate, retry those with fallback
	if c.Fallback == nil || c.Sample <= 0 || c.AdequacyRatio <= 0 {
		return res, nil
	}
	sampleCount := min(c.Sample, len(items))
	poor := make([]Item, 0, sampleCount)
	for i := range sampleCount {
		it := items[i]
		r, ok := res[it.ID]
		if !ok {
			continue
		}
		if isPoorAdequacy(it.Text, r.Translated, c.AdequacyRatio) {
			poor = append(poor, it)
		}
	}
	if len(poor) == 0 {
		return res, nil
	}
	fb, fbErr := c.Fallback.TranslateBatch(ctx, poor, target)
	if fbErr != nil {
		// keep primary results
		return res, nil
	}
	for _, it := range poor {
		pr := res[it.ID]
		fr, ok := fb[it.ID]
		if !ok {
			continue
		}
		// prefer fallback if it improved adequacy or produced non-empty when primary was empty
		if pr.Translated == "" || !isPoorAdequacy(it.Text, fr.Translated, c.AdequacyRatio) {
			res[it.ID] = fr
		}
	}
	return res, nil
}

func isPoorAdequacy(src, dst string, minRatio float64) bool {
	if len(src) == 0 {
		return false
	}
	// Empty translation is poor for non-English inputs; adequacy fallback logic will only run on items we chose to translate
	if len(dst) == 0 {
		return true
	}
	return float64(len(dst))/float64(len(src)) < minRatio
}
