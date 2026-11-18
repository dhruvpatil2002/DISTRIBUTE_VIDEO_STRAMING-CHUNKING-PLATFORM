package handlers

import (
    "encoding/json"
    "errors"
    "net/http"
    "strconv"
    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/auth"
    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/service"
)

type SiteHandler struct {
    Service *service.SiteService
}

func NewSiteHandler(
    svc *service.SiteService,
) *SiteHandler {
    return &SiteHandler{
        Service: svc,
    }
}

func writeJSON(
    w http.ResponseWriter,
    status int,
    data any,
) {
    w.Header().Set(
        "Content-Type",
        "application/json",
    )

    w.WriteHeader(status)

    _ = json.NewEncoder(w).Encode(data)
}

func (h *SiteHandler) Create(
    w http.ResponseWriter,
    r *http.Request,
) {

    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        Name     string  `json:"name"`
        Location *string `json:"location"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    site, err := h.Service.Create(
        r.Context(),
        userID,
        req.Name,
        req.Location,
    )

    if err != nil {
        if errors.Is(err, service.ErrInvalidInput) {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        http.Error(w, "Failed to create site", http.StatusInternalServerError)
        return
    }

    writeJSON(w, http.StatusCreated, site)
}

func (h *SiteHandler) GetAll(
    w http.ResponseWriter,
    r *http.Request,
) {

    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    limit := 10 // Set a default limit
    if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
        if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
            limit = l
        }
    }

    cursor := r.URL.Query().Get("cursor")

    sites, err := h.Service.GetAll(
        r.Context(),
        userID,
        limit,
        cursor,
    )
    

    if err != nil {
        http.Error(w, "Failed to fetch sites", http.StatusInternalServerError)
        return
    }

    writeJSON(w, http.StatusOK, sites)
}

func (h *SiteHandler) GetByID(
    w http.ResponseWriter,
    r *http.Request,
) {

    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    siteID := r.PathValue("siteID")

    site, err := h.Service.GetByID(
        r.Context(),
        siteID,
        userID,
    )

    if errors.Is(err, service.ErrNotFound) {
        http.Error(w, "Site not found", http.StatusNotFound)
        return
    }

    if err != nil {
        http.Error(w, "Failed to fetch site", http.StatusInternalServerError)
        return
    }

    writeJSON(w, http.StatusOK, site)
}

func (h *SiteHandler) Update(
    w http.ResponseWriter,
    r *http.Request,
) {

    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        Name     string  `json:"name"`
        Location *string `json:"location"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    site, err := h.Service.Update(
        r.Context(),
        r.PathValue("siteID"),
        userID,
        req.Name,
        req.Location,
    )

    if errors.Is(err, service.ErrNotFound) {
        http.Error(w, "Site not found", http.StatusNotFound)
        return
    }

    if errors.Is(err, service.ErrInvalidInput) {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if err != nil {
        http.Error(w, "Failed to update site", http.StatusInternalServerError)
        return
    }

    writeJSON(w, http.StatusOK, site)
}

func (h *SiteHandler) Delete(
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
        r.PathValue("siteID"),
        userID,
    )

    if errors.Is(err, service.ErrNotFound) {
        http.Error(w, "Site not found", http.StatusNotFound)
        return
    }

    if err != nil {
        http.Error(w, "Failed to delete site", http.StatusInternalServerError)
        return
    }

    writeJSON(w, http.StatusOK, map[string]string{
        "message": "Site deleted successfully",
    })
}