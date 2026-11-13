package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/models"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/repository"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/stream"
)

var (
	ErrPlaybackForbidden = errors.New(
		"playback session does not belong to user",
	)
)

type RTSPRelay interface {
	RegisterCamera(
		ctx context.Context,
		cameraID string,
		rtspURL string,
	) error

	UpdateCamera(
		ctx context.Context,
		cameraID string,
		rtspURL string,
	) error

	RemoveCamera(
		ctx context.Context,
		cameraID string,
	) error

	GetCameraRTSPURL(
		cameraID string,
	) (string, error)
}

type CameraService struct {
	Repo *repository.CameraRepository
	RTSP RTSPRelay
	HLS  *stream.HLSManager
}

func NewCameraService(
	repo *repository.CameraRepository,
	rtsp RTSPRelay,
	hls *stream.HLSManager,
) *CameraService {
	return &CameraService{
		Repo: repo,
		RTSP: rtsp,
		HLS:  hls,
	}
}

// validateRTSPURL validates the supplied camera URL.
func validateRTSPURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)

	if err != nil {
		return ErrInvalidInput
	}

	if parsed.Scheme != "rtsp" &&
		parsed.Scheme != "rtsps" {
		return ErrInvalidInput
	}

	if parsed.Hostname() == "" {
		return ErrInvalidInput
	}

	if parsed.User != nil {
		if _, ok := parsed.User.Password(); !ok {
			return ErrInvalidInput
		}
	}

	return nil
}

// Create registers a camera and its persistent RTSP relay.
func (s *CameraService) Create(
	ctx context.Context,
	siteID string,
	userID string,
	name string,
	rtspURL string,
) (*models.Camera, error) {

	name = strings.TrimSpace(name)
	rtspURL = strings.TrimSpace(rtspURL)

	if name == "" || len(name) > 150 {
		return nil, ErrInvalidInput
	}

	if len(rtspURL) > 4096 {
		return nil, ErrInvalidInput
	}

	if err := validateRTSPURL(rtspURL); err != nil {
		return nil, err
	}

	camera := &models.Camera{
		SiteID:  siteID,
		Name:    name,
		RTSPURL: rtspURL,
	}

	// 1. Save camera in PostgreSQL.
	if err := s.Repo.Create(
		ctx,
		camera,
		userID,
	); err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	// 2. Register persistent RTSP source in MediaMTX.
	if err := s.RTSP.RegisterCamera(
		ctx,
		camera.ID,
		camera.RTSPURL,
	); err != nil {

		// Compensate for failed relay registration.
		if rollbackErr := s.Repo.Delete(
			ctx,
			camera.ID,
			userID,
		); rollbackErr != nil {
			return nil, fmt.Errorf(
				"relay registration failed: %v; database rollback failed: %w",
				err,
				rollbackErr,
			)
		}

		return nil, fmt.Errorf(
			"failed to register camera relay: %w",
			err,
		)
	}

	return camera, nil
}

// GetAllBySite returns all cameras belonging to a site.
func (s *CameraService) GetAllBySite(
	ctx context.Context,
	siteID string,
	userID string,
) ([]models.Camera, error) {

	return s.Repo.GetAllBySite(
		ctx,
		siteID,
		userID,
	)
}

// GetByID returns a camera after ownership validation.
func (s *CameraService) GetByID(
	ctx context.Context,
	cameraID string,
	userID string,
) (*models.Camera, error) {

	camera, err := s.Repo.GetByID(
		ctx,
		cameraID,
		userID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	return camera, err
}

// Update updates camera information.
func (s *CameraService) Update(
	ctx context.Context,
	cameraID string,
	userID string,
	name string,
) (*models.Camera, error) {

	name = strings.TrimSpace(name)

	if name == "" || len(name) > 150 {
		return nil, ErrInvalidInput
	}

	camera := &models.Camera{
		ID:   cameraID,
		Name: name,
	}

	if err := s.Repo.Update(
		ctx,
		camera,
		userID,
	); err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return camera, nil
}

// Delete removes the relay and camera record.
func (s *CameraService) Delete(
	ctx context.Context,
	cameraID string,
	userID string,
) error {

	camera, err := s.Repo.GetByID(
		ctx,
		cameraID,
		userID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	if err != nil {
		return err
	}

	// Remove MediaMTX path first.
	if err := s.RTSP.RemoveCamera(
		ctx,
		camera.ID,
	); err != nil {
		return fmt.Errorf(
			"failed to remove camera relay: %w",
			err,
		)
	}

	// Remove camera from PostgreSQL.
	if err := s.Repo.Delete(
		ctx,
		cameraID,
		userID,
	); err != nil {
		return err
	}

	return nil
}

// StartPlayback starts or reuses the camera's HLS process.
func (s *CameraService) StartPlayback(
	ctx context.Context,
	cameraID string,
	userID string,
) (string, string, error) {

	// 1. Verify camera ownership.
	camera, err := s.Repo.GetByID(
		ctx,
		cameraID,
		userID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrNotFound
		}

		return "", "", err
	}

	// 2. Get internal MediaMTX RTSP URL.
	relayURL, err := s.RTSP.GetCameraRTSPURL(
		camera.ID,
	)

	if err != nil {
		return "", "", err
	}

	// 3. Acquire playback lease.
	leaseID, playlistURL, err := s.HLS.Acquire(
		camera.ID,
		relayURL,
		userID,
	)

	if err != nil {
		return "", "", err
	}

	// 4. Wait for FFmpeg readiness.
	if err := s.HLS.WaitUntilReady(
		ctx,
		camera.ID,
	); err != nil {

		s.HLS.Release(leaseID)

		return "", "", err
	}

	return leaseID, playlistURL, nil
}

// StopPlayback releases a viewer lease.
func (s *CameraService) StopPlayback(
	ctx context.Context,
	leaseID string,
	userID string,
) error {

	s.HLS.Release(leaseID)
	return nil
}

// Heartbeat renews an active playback lease.
func (s *CameraService) Heartbeat(
	ctx context.Context,
	leaseID string,
	userID string,
) error {

	return s.HLS.Heartbeat(
		leaseID,
		userID,
	)
}
