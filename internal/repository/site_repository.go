package repository

import (
    "context"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/models"
)

type SiteRepository struct {
    DB *pgxpool.Pool
}

func NewSiteRepository(
    db *pgxpool.Pool,
) *SiteRepository {
    return &SiteRepository{
        DB: db,
    }
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
) ([]models.Site, error) {

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
        ORDER BY created_at DESC
    `

    rows, err := r.DB.Query(
        ctx,
        query,
        userID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    sites := make([]models.Site, 0)

    for rows.Next() {
        var site models.Site

        err := rows.Scan(
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

        sites = append(sites, site)
    }

    return sites, rows.Err()
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