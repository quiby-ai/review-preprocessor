package storage

import (
	"context"
	"database/sql"
	"time"
)

type CleanRepository struct{ db *sql.DB }

func NewCleanRepository(db *sql.DB) *CleanRepository { return &CleanRepository{db: db} }

type CleanReview struct {
	ID                   string
	AppID                string
	Country              string
	Rating               int16
	Title                string
	ContentClean         string
	Language             string
	ContentEN            *string
	IsContentful         bool
	ReviewedAt           time.Time
	ResponseDate         *time.Time
	ResponseContentClean *string
}

func (r *CleanRepository) UpsertBatch(ctx context.Context, items []CleanReview) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO clean_reviews (id, app_id, country, rating, title, content_clean, language, content_en, is_contentful, reviewed_at, response_date, response_content_clean)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET
			app_id = EXCLUDED.app_id,
			country = EXCLUDED.country,
			rating = EXCLUDED.rating,
			title = EXCLUDED.title,
			content_clean = EXCLUDED.content_clean,
            language = EXCLUDED.language,
            content_en = EXCLUDED.content_en,
            is_contentful = EXCLUDED.is_contentful,
			reviewed_at = EXCLUDED.reviewed_at,
			response_date = EXCLUDED.response_date,
			response_content_clean = EXCLUDED.response_content_clean`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, it := range items {
		_, err := stmt.ExecContext(ctx, it.ID, it.AppID, it.Country, it.Rating, it.Title, it.ContentClean, it.Language, it.ContentEN, it.IsContentful, it.ReviewedAt, it.ResponseDate, it.ResponseContentClean)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
