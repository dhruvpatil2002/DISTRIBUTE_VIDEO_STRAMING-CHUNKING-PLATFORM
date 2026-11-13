package service

import (
    "context"
    "errors"
    "strings"

    "github.com/jackc/pgx/v5"

    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/models"
    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/repository"
)

var (
    ErrInvalidInput = errors.New("invalid input")
    ErrNotFound     = errors.New("resource not found")
)

type SiteService struct {
    Repo *repository.SiteRepository
}

func NewSiteService(
    repo *repository.SiteRepository,
) *SiteService {
    return &SiteService{
        Repo: repo,
    }
}

func (s *SiteService) Create(
    ctx context.Context,
    userID string,
    name string,
    location *string,
) (*models.Site, error) {

    name = strings.TrimSpace(name)

    if name == "" || len(name) > 150 {
        return nil, ErrInvalidInput
    }

    if location != nil {
        value := strings.TrimSpace(*location)

        if len(value) > 255 {
            return nil, ErrInvalidInput
        }

        location = &value
    }

    site := &models.Site{
        UserID:   userID,
        Name:     name,
        Location: location,
    }

    if err := s.Repo.Create(ctx, site); err != nil {
        return nil, err
    }

    return site, nil
}

func (s *SiteService) GetAll(
    ctx context.Context,
    userID string,
) ([]models.Site, error) {

    return s.Repo.GetAll(ctx, userID)
}

func (s *SiteService) GetByID(
    ctx context.Context,
    siteID string,
    userID string,
) (*models.Site, error) {

    site, err := s.Repo.GetByID(
        ctx,
        siteID,
        userID,
    )

    if errors.Is(err, pgx.ErrNoRows) {
        return nil, ErrNotFound
    }

    return site, err
}

func (s *SiteService) Update(
    ctx context.Context,
    siteID string,
    userID string,
    name string,
    location *string,
) (*models.Site, error) {

    name = strings.TrimSpace(name)

    if name == "" || len(name) > 150 {
        return nil, ErrInvalidInput
    }

    if location != nil {
        value := strings.TrimSpace(*location)

        if len(value) > 255 {
            return nil, ErrInvalidInput
        }

        location = &value
    }

    site := &models.Site{
        ID:       siteID,
        UserID:   userID,
        Name:     name,
        Location: location,
    }

    if err := s.Repo.Update(ctx, site); err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrNotFound
        }

        return nil, err
    }

    return site, nil
}

func (s *SiteService) Delete(
    ctx context.Context,
    siteID string,
    userID string,
) error {

    err := s.Repo.Delete(ctx, siteID, userID)

    if errors.Is(err, pgx.ErrNoRows) {
        return ErrNotFound
    }

    return err
}