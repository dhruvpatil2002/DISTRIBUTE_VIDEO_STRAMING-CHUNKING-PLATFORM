package handlers

import (
	"context"
	"net/http"
	"time"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

func Health() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "video-platform-api",
		})
	}
}

func Ready(db DBPinger) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(
			r.Context(),
			3*time.Second,
		)
		defer cancel()

		if err := db.Ping(ctx); err != nil {

			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "not_ready",
				"error":  "database unavailable",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":   "ready",
			"database": "connected",
		})
	}
}

