
package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/models"
)

var ErrInvalidCursor = errors.New("invalid site cursor")

type SiteRepository struct {
	DB *pgxpool.Pool
}

func NewSiteRepository(db *pgxpool.Pool) *SiteRepository {
	return &SiteRepository{
		DB: db,
	}
}

type SiteCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

type SitePage struct {
	Sites      []models.Site `json:"sites"`
	NextCursor string        `json:"next_cursor,omitempty"`
	HasMore    bool          `json:"has_more"`
}

func EncodeSiteCursor(c SiteCursor) (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

func DecodeSiteCursor(value string) (*SiteCursor, error) {
	if value == "" {
		return nil, nil
	}

	// Prevent excessively large cursor input.
	if len(value) > 1024 {
		return nil, ErrInvalidCursor
	}

	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	var cursor SiteCursor

	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, ErrInvalidCursor
	}

	if cursor.ID == "" || cursor.CreatedAt.IsZero() {
		return nil, ErrInvalidCursor
	}

	return &cursor, nil
}

func (r *SiteRepository) Create(
	ctx context.Context,
	site *models.Site,
) error {
	query := `
		INSERT INTO sites (
			user_id,
			name,
			location
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			created_at,
			updated_at
	`

	return r.DB.QueryRow(
		ctx,
		query,
		site.UserID,
		site.Name,
		site.Location,
	).Scan(
		&site.ID,
		&site.CreatedAt,
		&site.UpdatedAt,
	)
}

func (r *SiteRepository) GetAll(
	ctx context.Context,
	userID string,
	limit int,
	cursor *SiteCursor,
) (*SitePage, error) {

	if limit < 1 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	var cursorTime *time.Time
	var cursorID *string

	if cursor != nil {
		cursorTime = &cursor.CreatedAt
		cursorID = &cursor.ID
	}

	query := `
		SELECT
			id,
			user_id,
			name,
			location,
			created_at,
			updated_at
		FROM sites
		WHERE user_id = $1
		  AND (
			  $2::timestamptz IS NULL
			  OR (created_at, id) <
				 ($2::timestamptz, $3::uuid)
		  )
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`

	rows, err := r.DB.Query(
		ctx,
		query,
		userID,
		cursorTime,
		cursorID,
		limit+1,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sites := make([]models.Site, 0, limit+1)

	for rows.Next() {
		var site models.Site

		if err := rows.Scan(
			&site.ID,
			&site.UserID,
			&site.Name,
			&site.Location,
			&site.CreatedAt,
			&site.UpdatedAt,
		); err != nil {
			return nil, err
		}

		sites = append(sites, site)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(sites) > limit

	if hasMore {
		sites = sites[:limit]
	}

	result := &SitePage{
		Sites:   sites,
		HasMore: hasMore,
	}

	if hasMore {
		last := sites[len(sites)-1]

		nextCursor, err := EncodeSiteCursor(SiteCursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		})
		if err != nil {
			return nil, err
		}

		result.NextCursor = nextCursor
	}

	return result, nil
}

func (r *SiteRepository) GetByID(
	ctx context.Context,
	id string,
	userID string,
) (*models.Site, error) {

	query := `
		SELECT
			id,
			user_id,
			name,
			location,
			created_at,
			updated_at
		FROM sites
		WHERE id = $1 AND user_id = $2
	`

	var site models.Site

	err := r.DB.QueryRow(
		ctx,
		query,
		id,
		userID,
	).Scan(
		&site.ID,
		&site.UserID,
		&site.Name,
		&site.Location,
		&site.CreatedAt,
		&site.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &site, nil
}

func (r *SiteRepository) Update(
	ctx context.Context,
	site *models.Site,
) error {

	query := `
		UPDATE sites
		SET
			name = $1,
			location = $2,
			updated_at = NOW()
		WHERE id = $3 AND user_id = $4
		RETURNING updated_at
	`

	return r.DB.QueryRow(
		ctx,
		query,
		site.Name,
		site.Location,
		site.ID,
		site.UserID,
	).Scan(&site.UpdatedAt)
}

func (r *SiteRepository) Delete(
	ctx context.Context,
	id string,
	userID string,
) error {

	query := `
		DELETE FROM sites
		WHERE id = $1 AND user_id = $2
	`

	result, err := r.DB.Exec(
		ctx,
		query,
		id,
		userID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
