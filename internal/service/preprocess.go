package service

import (
	"context"
	"time"

	"github.com/quiby-ai/review-preprocessor/config"
	"github.com/quiby-ai/review-preprocessor/internal/producer"
	"github.com/quiby-ai/review-preprocessor/internal/storage"
	"github.com/quiby-ai/review-preprocessor/internal/textutil"
)

type PrepareReviewsEvent struct {
	JobID     string   `json:"job_id"`
	AppID     string   `json:"app_id"`
	Countries []string `json:"countries"`
	DateFrom  string   `json:"date_from"` // RFC3339 or YYYY-MM-DD
	DateTo    string   `json:"date_to"`
	Limit     int      `json:"limit"`
}

type PreprocessService struct {
	raw   *storage.RawRepository
	clean *storage.CleanRepository
	prod  *producer.KafkaProducer
	cfg   config.ProcessingConfig
}

func NewPreprocessService(raw *storage.RawRepository, clean *storage.CleanRepository, prod *producer.KafkaProducer, cfg config.ProcessingConfig) *PreprocessService {
	return &PreprocessService{raw: raw, clean: clean, prod: prod, cfg: cfg}
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

func (s *PreprocessService) Handle(ctx context.Context, evt PrepareReviewsEvent) error {
	from := parseTime(evt.DateFrom, time.Time{})
	to := parseTime(evt.DateTo, time.Now().UTC())
	if evt.Limit <= 0 {
		evt.Limit = s.cfg.BatchSize
	}

	rawItems, err := s.raw.Fetch(ctx, storage.RawFilters{
		AppID:     evt.AppID,
		Countries: evt.Countries,
		DateFrom:  from,
		DateTo:    to,
		Limit:     evt.Limit,
	})
	if err != nil {
		return err
	}

	cleanBatch := make([]storage.CleanReview, 0, len(rawItems))
	ids := make([]string, 0, len(rawItems))
	for _, rr := range rawItems {
		cleanText, ok := textutil.Clean(rr.Content, s.cfg.HTMLStrip, s.cfg.EmojiStrip, s.cfg.WhitespaceNormalize, s.cfg.MaxReviewLen, s.cfg.MinContentLen)
		if !ok {
			continue
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
			Language:             s.cfg.DefaultLang, // TODO: auto-detect/translate later
			ReviewedAt:           rr.ReviewedAt,
			ResponseDate:         respDate,
			ResponseContentClean: respTextClean,
		})
		ids = append(ids, rr.ID)
	}

	if err := s.clean.UpsertBatch(ctx, cleanBatch); err != nil {
		return err
	}

	if s.cfg.PublishIDsLimit > 0 && len(ids) > s.cfg.PublishIDsLimit {
		ids = ids[:s.cfg.PublishIDsLimit]
	}
	return s.prod.PublishClusterEvent(ctx, producer.ClusterReviewsEvent{
		JobID: evt.JobID,
		AppID: evt.AppID,
		IDs:   ids,
		Count: len(cleanBatch),
	})
}
