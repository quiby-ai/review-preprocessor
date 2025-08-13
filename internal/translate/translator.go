package translate

import "context"

type Item struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Result struct {
	ID         string `json:"id"`
	Lang       string `json:"lang"`
	Translated string `json:"translated"`
}

type Translator interface {
	TranslateBatch(ctx context.Context, items []Item, target string) (map[string]Result, error)
}

// Noop translator returns empty translations.
type Noop struct{}

func (n Noop) TranslateBatch(ctx context.Context, items []Item, target string) (map[string]Result, error) {
	out := make(map[string]Result, len(items))
	for _, it := range items {
		out[it.ID] = Result{ID: it.ID, Lang: "und", Translated: ""}
	}
	return out, nil
}
