package handlers

import (
    "encoding/json"
    "errors"
    "net/http"

    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/auth"
    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/service"
)

type CameraHandler struct {
    Service *service.CameraService
}

func NewCameraHandler(
    svc *service.CameraService,
) *CameraHandler {
    return &CameraHandler{
        Service: svc,
    }
}

func (h *CameraHandler) Create(
    w http.ResponseWriter,
    r *http.Request,
) {

    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        Name    string `json:"name"`
        RTSPURL string `json:"rtsp_url"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    camera, err := h.Service.Create(
        r.Context(),
        r.PathValue("siteID"),
        userID,
        req.Name,
        req.RTSPURL,
    )

    if errors.Is(err, service.ErrInvalidInput) {
        http.Error(w, "Invalid camera details", http.StatusBadRequest)
        return
    }

    if errors.Is(err, service.ErrNotFound) {
        http.Error(w, "Site not found", http.StatusNotFound)
        return
    }

    if err != nil {
        http.Error(w, "Failed to create camera", http.StatusInternalServerError)
        return
    }

    writeJSON(w, http.StatusCreated, camera)
}

func (h *CameraHandler) GetAllBySite(
    w http.ResponseWriter,
    r *http.Request,
) {

    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    cameras, err := h.Service.GetAllBySite(
        r.Context(),
        r.PathValue("siteID"),
        userID,
    )

    if err != nil {
        http.Error(w, "Failed to fetch cameras", http.StatusInternalServerError)
        return
    }

    writeJSON(w, http.StatusOK, cameras)
}

func (h *CameraHandler) GetByID(
    w http.ResponseWriter,
    r *http.Request,
) {

    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    camera, err := h.Service.GetByID(
        r.Context(),
        r.PathValue("cameraID"),
        userID,
    )

    if errors.Is(err, service.ErrNotFound) {
        http.Error(w, "Camera not found", http.StatusNotFound)
        return
    }

    if err != nil {
        http.Error(w, "Failed to fetch camera", http.StatusInternalServerError)
        return
    }

    writeJSON(w, http.StatusOK, camera)
}

func (h *CameraHandler) Update(
    w http.ResponseWriter,
    r *http.Request,
) {

    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        Name string `json:"name"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    camera, err := h.Service.Update(
        r.Context(),
        r.PathValue("cameraID"),
        userID,
        req.Name,
    )

    if errors.Is(err, service.ErrNotFound) {
        http.Error(w, "Camera not found", http.StatusNotFound)
        return
    }

    if errors.Is(err, service.ErrInvalidInput) {
        http.Error(w, "Invalid camera name", http.StatusBadRequest)
        return
    }

    if err != nil {
        http.Error(w, "Failed to update camera", http.StatusInternalServerError)
        return
    }

    writeJSON(w, http.StatusOK, camera)
}

func (h *CameraHandler) Delete(
    w http.ResponseWriter,
    r *http.Request,
) {

    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    err := h.Service.Delete(
        r.Context(),
        r.PathValue("cameraID"),
        userID,
    )

    if errors.Is(err, service.ErrNotFound) {
        http.Error(w, "Camera not found", http.StatusNotFound)
        return
    }

    if err != nil {
        http.Error(w, "Failed to delete camera", http.StatusInternalServerError)
        return
    }

    writeJSON(w, http.StatusOK, map[string]string{
        "message": "Camera deleted successfully",
    })
}