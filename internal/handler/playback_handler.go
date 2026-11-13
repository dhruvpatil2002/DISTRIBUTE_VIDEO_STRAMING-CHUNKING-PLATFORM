
package handlers

import (
	"errors"
	"net/http"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/auth"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/service"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/stream"
)

type PlaybackHandler struct {
	Service *service.CameraService
}

func NewPlaybackHandler(
	svc *service.CameraService,
) *PlaybackHandler {
	return &PlaybackHandler{
		Service: svc,
	}
}

func (h *PlaybackHandler) Start(
	w http.ResponseWriter,
	r *http.Request,
) {

	userID, ok := auth.GetUserID(r.Context())

	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	cameraID := r.PathValue("cameraID")

	leaseID, playlistURL, err := h.Service.StartPlayback(
		r.Context(),
		cameraID,
		userID,
	)

	if err != nil {
		http.Error(
			w,
			"Unable to start playback",
			http.StatusBadGateway,
		)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"lease_id":     leaseID,
		"playlist_url": playlistURL,
		"status":       "starting",
	})
}

func (h *PlaybackHandler) Stop(
	w http.ResponseWriter,
	r *http.Request,
) {

	userID, ok := auth.GetUserID(r.Context())

	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	leaseID := r.PathValue("leaseID")

	if err := h.Service.StopPlayback(
		r.Context(),
		leaseID,
		userID,
	); err != nil {

		if errors.Is(err, stream.ErrLeaseExpired) {
			http.Error(w, "Playback session expired", http.StatusGone)
			return
		}

		http.Error(
			w,
			"Unable to stop playback",
			http.StatusInternalServerError,
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Playback session released",
	})
}

func (h *PlaybackHandler) Heartbeat(
	w http.ResponseWriter,
	r *http.Request,
) {

	userID, ok := auth.GetUserID(r.Context())

	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	leaseID := r.PathValue("leaseID")

	err := h.Service.Heartbeat(
		r.Context(),
		leaseID,
		userID,
	)

	if errors.Is(err, stream.ErrLeaseExpired) {
		http.Error(
			w,
			"Playback session expired",
			http.StatusGone,
		)
		return
	}

	if errors.Is(err, service.ErrPlaybackForbidden) {
		http.Error(
			w,
			"Playback session does not belong to user",
			http.StatusForbidden,
		)
		return
	}

	if err != nil {
		http.Error(
			w,
			"Heartbeat failed",
			http.StatusInternalServerError,
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "active",
	})
}
