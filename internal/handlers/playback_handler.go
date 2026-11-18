
package handlers

import (
    "context"
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
    return &PlaybackHandler{Service: svc}
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
    if cameraID == "" {
        http.Error(w, "Camera ID required", http.StatusBadRequest)
        return
    }

    leaseID, playlistURL, err := h.Service.StartPlayback(
        r.Context(),
        cameraID,
        userID,
    )

    if err != nil {
        switch {
        case errors.Is(err, service.ErrNotFound):
            http.Error(w, "Camera not found", http.StatusNotFound)

        case errors.Is(err, context.Canceled):
            // Client disconnected or request was cancelled.
            return

        case errors.Is(err, context.DeadlineExceeded):
            http.Error(w, "Playback startup timeout", http.StatusGatewayTimeout)

        case errors.Is(err, stream.ErrStreamFailed):
            http.Error(w, "Stream unavailable", http.StatusBadGateway)

        default:
            http.Error(w, "Unable to start playback", http.StatusBadGateway)
        }
        return
    }

    writeJSON(w, http.StatusOK, map[string]string{
        "lease_id":     leaseID,
        "playlist_url": playlistURL,
        "status":       "ready",
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
    if leaseID == "" {
        http.Error(w, "Lease ID required", http.StatusBadRequest)
        return
    }

    err := h.Service.StopPlayback(
        r.Context(),
        leaseID,
        userID,
    )

    switch {
    case err == nil:
        writeJSON(w, http.StatusOK, map[string]string{
            "message": "Playback stopped",
        })

    case errors.Is(err, stream.ErrLeaseForbidden):
        http.Error(w, "Forbidden", http.StatusForbidden)

    case errors.Is(err, stream.ErrLeaseExpired):
        http.Error(w, "Playback lease expired", http.StatusGone)

    default:
        http.Error(w, "Unable to stop playback", http.StatusInternalServerError)
    }
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
    if leaseID == "" {
        http.Error(w, "Lease ID required", http.StatusBadRequest)
        return
    }

    err := h.Service.Heartbeat(
        r.Context(),
        leaseID,
        userID,
    )

    switch {
    case err == nil:
        writeJSON(w, http.StatusOK, map[string]string{
            "status": "active",
        })

    case errors.Is(err, stream.ErrLeaseForbidden):
        http.Error(w, "Forbidden", http.StatusForbidden)

    case errors.Is(err, stream.ErrLeaseExpired):
        http.Error(w, "Playback lease expired", http.StatusGone)

    default:
        http.Error(w, "Heartbeat failed", http.StatusInternalServerError)
    }
}
