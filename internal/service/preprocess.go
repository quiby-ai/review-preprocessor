package service

import (
	"context"
	"time"

	"github.com/quiby-ai/common/pkg/events"
	"github.com/quiby-ai/review-preprocessor/config"
	"github.com/quiby-ai/review-preprocessor/internal/lang"
	"github.com/quiby-ai/review-preprocessor/internal/producer"
	"github.com/quiby-ai/review-preprocessor/internal/storage"
	"github.com/quiby-ai/review-preprocessor/internal/textutil"
	"github.com/quiby-ai/review-preprocessor/internal/translate"
)

type PreprocessService struct {
	raw   *storage.RawRepository
	clean *storage.CleanRepository
	prod  *producer.Producer
	cfg   config.ProcessingConfig
	tr    translate.Translator
}

func NewPreprocessService(raw *storage.RawRepository, clean *storage.CleanRepository, prod *producer.Producer, cfg config.ProcessingConfig, tr translate.Translator) *PreprocessService {
	if tr == nil {
		tr = translate.Noop{}
	}
	return &PreprocessService{raw: raw, clean: clean, prod: prod, cfg: cfg, tr: tr}
}

func parseTime(s string, def time.Time) time.Time {
	if s == "" {
		return def
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return def
}

func (s *PreprocessService) Handle(ctx context.Context, evt events.PrepareRequest, sagaID string) error {
	from := parseTime(evt.DateFrom, time.Time{})
	to := parseTime(evt.DateTo, time.Now().UTC())

	rawItems, err := s.raw.Fetch(ctx, storage.RawFilters{
		AppID:     evt.AppID,
		Countries: evt.Countries,
		DateFrom:  from,
		DateTo:    to,
	})
	if err != nil {
		return err
	}

	cleanBatch, ids := s.buildCleanBatch(rawItems)
	s.runTranslations(ctx, &cleanBatch)

	if err := s.clean.UpsertBatch(ctx, cleanBatch); err != nil {
		return err
	}

	if s.cfg.PublishIDsLimit > 0 && len(ids) > s.cfg.PublishIDsLimit {
		ids = ids[:s.cfg.PublishIDsLimit]
	}
	prepareCompleted := events.PrepareCompleted{
		PrepareRequest: evt,
		CleanCount:     len(ids),
	}
	envelope := s.prod.BuildEnvelope(prepareCompleted, sagaID)
	return s.prod.PublishEvent(ctx, []byte(sagaID), envelope)
}

// buildCleanBatch cleans, checks contentfulness, detects language, and builds the batch.
// It also determines which IDs to publish (contentful only) and which items require translation.
func (s *PreprocessService) buildCleanBatch(rawItems []storage.RawReview) ([]storage.CleanReview, []string) {
	cleanBatch := make([]storage.CleanReview, 0, len(rawItems))
	ids := make([]string, 0, len(rawItems))
	for _, rr := range rawItems {
		cleanText, ok := textutil.Clean(rr.Content, s.cfg.HTMLStrip, s.cfg.EmojiStrip, s.cfg.WhitespaceNormalize, s.cfg.MaxReviewLen, s.cfg.MinContentLen)
		if !ok {
			if s.cfg.SaveSkipped {
				cleanBatch = append(cleanBatch, s.skippedClean(rr, cleanText))
			}
			continue
		}
		if !textutil.IsContentful(cleanText, s.cfg.MinWords, s.cfg.MinChars, s.cfg.MinAlphaRatio) {
			if s.cfg.SaveSkipped {
				cleanBatch = append(cleanBatch, s.skippedClean(rr, cleanText))
			}
			continue
		}
		langCode, conf := lang.DetectCode(cleanText)
		lowConf := langCode == "und" || conf < s.cfg.LangDetectMinConf
		if lowConf && langCode == "und" {
			langCode = s.cfg.DefaultLang
		}
		var respTextClean *string
		if rr.ResponseContent.Valid {
			if v, ok := textutil.Clean(rr.ResponseContent.String, s.cfg.HTMLStrip, s.cfg.EmojiStrip, s.cfg.WhitespaceNormalize, s.cfg.MaxReviewLen, s.cfg.MinContentLen); ok {
				respTextClean = &v
			}
		}
		var respDate *time.Time
		if rr.ResponseDate.Valid {
			d := rr.ResponseDate.Time
			respDate = &d
		}
		cleanBatch = append(cleanBatch, storage.CleanReview{
			ID:                   rr.ID,
			AppID:                rr.AppID,
			Country:              rr.Country,
			Rating:               rr.Rating,
			Title:                rr.Title,
			ContentClean:         cleanText,
			Language:             langCode,
			IsContentful:         true,
			ReviewedAt:           rr.ReviewedAt,
			ResponseDate:         respDate,
			ResponseContentClean: respTextClean,
		})
		ids = append(ids, rr.ID)
	}
	return cleanBatch, ids
}

func (s *PreprocessService) skippedClean(rr storage.RawReview, cleanText string) storage.CleanReview {
	return storage.CleanReview{
		ID:           rr.ID,
		AppID:        rr.AppID,
		Country:      rr.Country,
		Rating:       rr.Rating,
		Title:        rr.Title,
		ContentClean: cleanText,
		Language:     s.cfg.DefaultLang,
		IsContentful: false,
		ReviewedAt:   rr.ReviewedAt,
	}
}

// runTranslations translates only items that are non-EN or had low-confidence detection.
func (s *PreprocessService) runTranslations(ctx context.Context, batch *[]storage.CleanReview) {
	if !s.cfg.TranslateEnabled || s.cfg.TranslateTargetLang != "en" {
		return
	}
	toTranslate := make([]translate.Item, 0)
	idToIndex := make(map[string]int)
	for i := range *batch {
		b := &(*batch)[i]
		// Skip non-contentful
		if !b.IsContentful {
			continue
		}
		if b.Language != "en" {
			toTranslate = append(toTranslate, translate.Item{ID: b.ID, Text: b.ContentClean})
			idToIndex[b.ID] = i
		}
	}
	if len(toTranslate) == 0 {
		return
	}
	bs := s.cfg.TranslateBatchSize
	if bs <= 0 {
		bs = 20
	}
	for i := 0; i < len(toTranslate); i += bs {
		j := i + bs
		if j > len(toTranslate) {
			j = len(toTranslate)
		}
		sub := toTranslate[i:j]
		tctx := ctx
		if s.cfg.TranslateTimeout > 0 {
			var cancel context.CancelFunc
			tctx, cancel = context.WithTimeout(ctx, s.cfg.TranslateTimeout)
			defer cancel()
		}
		res, err := s.tr.TranslateBatch(tctx, sub, s.cfg.TranslateTargetLang)
		if err != nil {
			continue
		}
		for _, it := range sub {
			r, ok := res[it.ID]
			if !ok {
				continue
			}
			idx := idToIndex[it.ID]
			if r.Translated != "" {
				en := r.Translated
				(*batch)[idx].ContentEN = &en
			}
			if r.Lang != "" {
				(*batch)[idx].Language = r.Lang
			}
		}
	}
}
