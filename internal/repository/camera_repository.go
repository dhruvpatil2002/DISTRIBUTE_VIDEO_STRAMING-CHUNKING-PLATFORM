package repository

import (
    "context"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/constants"
    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/models"
)

type CameraRepository struct {
    DB *pgxpool.Pool
}

func NewCameraRepository(
    db *pgxpool.Pool,
) *CameraRepository {
    return &CameraRepository{
        DB: db,
    }
}

func (r *CameraRepository) Create(
    ctx context.Context,
    camera *models.Camera,
    userID string,
) error {

    query := `
        INSERT INTO cameras (
            site_id,
            name,
            rtsp_url,
            status
        )
        SELECT
            s.id,
            $3,
            $4,
            $5
        FROM sites s
        WHERE s.id = $1
          AND s.user_id = $2
        RETURNING
            id,
            created_at,
            updated_at
    `

    return r.DB.QueryRow(
        ctx,
        query,
        camera.SiteID,
        userID,
        camera.Name,
        camera.RTSPURL,
        constants.CameraStatusRegistered,
    ).Scan(
        &camera.ID,
        &camera.CreatedAt,
        &camera.UpdatedAt,
    )
}

func (r *CameraRepository) GetAllBySite(
    ctx context.Context,
    siteID string,
    userID string,
) ([]models.Camera, error) {

    query := `
        SELECT
            c.id,
            c.site_id,
            c.name,
            c.status,
            c.hls_url,
            c.created_at,
            c.updated_at
        FROM cameras c
        INNER JOIN sites s
            ON s.id = c.site_id
        WHERE c.site_id = $1
          AND s.user_id = $2
        ORDER BY c.created_at DESC
    `

    rows, err := r.DB.Query(
        ctx,
        query,
        siteID,
        userID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    cameras := make([]models.Camera, 0)

    for rows.Next() {
        var camera models.Camera

        err := rows.Scan(
            &camera.ID,
            &camera.SiteID,
            &camera.Name,
            &camera.Status,
            &camera.HLSURL,
            &camera.CreatedAt,
            &camera.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }

        cameras = append(cameras, camera)
    }

    return cameras, rows.Err()
}

func (r *CameraRepository) GetByID(
    ctx context.Context,
    cameraID string,
    userID string,
) (*models.Camera, error) {

    query := `
        SELECT
            c.id,
            c.site_id,
            c.name,
            c.rtsp_url,
            c.status,
            c.hls_url,
            c.created_at,
            c.updated_at
        FROM cameras c
        INNER JOIN sites s
            ON s.id = c.site_id
        WHERE c.id = $1
          AND s.user_id = $2
    `

    var camera models.Camera

    err := r.DB.QueryRow(
        ctx,
        query,
        cameraID,
        userID,
    ).Scan(
        &camera.ID,
        &camera.SiteID,
        &camera.Name,
        &camera.RTSPURL,
        &camera.Status,
        &camera.HLSURL,
        &camera.CreatedAt,
        &camera.UpdatedAt,
    )

    if err != nil {
        return nil, err
    }

    return &camera, nil
}

func (r *CameraRepository) Update(
    ctx context.Context,
    camera *models.Camera,
    userID string,
) error {

    query := `
        UPDATE cameras c
        SET
            name = $1,
            updated_at = NOW()
        FROM sites s
        WHERE c.site_id = s.id
          AND c.id = $2
          AND s.user_id = $3
        RETURNING c.updated_at
    `

    return r.DB.QueryRow(
        ctx,
        query,
        camera.Name,
        camera.ID,
        userID,
    ).Scan(&camera.UpdatedAt)
}

func (r *CameraRepository) UpdateStatus(
    ctx context.Context,
    cameraID string,
    status string,
) error {

    query := `
        UPDATE cameras
        SET
            status = $1,
            updated_at = NOW()
        WHERE id = $2
    `

    result, err := r.DB.Exec(
        ctx,
        query,
        status,
        cameraID,
    )
    if err != nil {
        return err
    }

    if result.RowsAffected() == 0 {
        return pgx.ErrNoRows
    }

    return nil
}

func (r *CameraRepository) Delete(
    ctx context.Context,
    cameraID string,
    userID string,
) error {

    query := `
        DELETE FROM cameras c
        USING sites s
        WHERE c.site_id = s.id
          AND c.id = $1
          AND s.user_id = $2
    `

    result, err := r.DB.Exec(
        ctx,
        query,
        cameraID,
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

func (r *CameraRepository) GetAllForStreaming(
    ctx context.Context,
) ([]models.Camera, error) {

    query := `
        SELECT id, rtsp_url
        FROM cameras
    `

    rows, err := r.DB.Query(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    cameras := make([]models.Camera, 0)

    for rows.Next() {
        var camera models.Camera

        if err := rows.Scan(
            &camera.ID,
            &camera.RTSPURL,
        ); err != nil {
            return nil, err
        }

        cameras = append(cameras, camera)
    }

    return cameras, rows.Err()
}