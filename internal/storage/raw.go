package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type RawRepository struct{ db *sql.DB }

func NewRawRepository(db *sql.DB) *RawRepository { return &RawRepository{db: db} }

type RawFilters struct {
	AppID     string
	Countries []string
	DateFrom  time.Time
	DateTo    time.Time
}

type RawReview struct {
	ID              string
	AppID           string
	Country         string
	Rating          int16
	Title           string
	Content         string
	ReviewedAt      time.Time
	ResponseDate    sql.NullTime
	ResponseContent sql.NullString
}

func (r *RawRepository) Fetch(ctx context.Context, f RawFilters) ([]RawReview, error) {
	q := `SELECT id, app_id, country, rating, title, content, reviewed_at, response_date, response_content
		FROM raw_reviews
		WHERE app_id = $1
		AND ($2::text[] IS NULL OR country = ANY($2))
		AND reviewed_at >= $3 AND reviewed_at <= $4
		ORDER BY reviewed_at ASC`

	var rows *sql.Rows
	var err error
	if len(f.Countries) == 0 {
		rows, err = r.db.QueryContext(ctx, q, f.AppID, nil, f.DateFrom, f.DateTo)
	} else {
		rows, err = r.db.QueryContext(ctx, q, f.AppID, pq.Array(f.Countries), f.DateFrom, f.DateTo)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RawReview{}
	for rows.Next() {
		var rr RawReview
		if err := rows.Scan(&rr.ID, &rr.AppID, &rr.Country, &rr.Rating, &rr.Title, &rr.Content, &rr.ReviewedAt, &rr.ResponseDate, &rr.ResponseContent); err != nil {
			return nil, err
		}
		out = append(out, rr)
	}
	return out, rows.Err()
}
