
package models

import "time"

type Video struct {
	ID                string    `json:"id"`
	UserID            *string   `json:"user_id,omitempty"`
	Title             string    `json:"title"`
	OriginalFilename  string    `json:"original_filename"`
	OriginalURL       *string   `json:"original_url,omitempty"`
	MasterPlaylistURL *string   `json:"master_playlist_url,omitempty"`
	MimeType          *string   `json:"mime_type,omitempty"`
	FileSizeBytes     int64     `json:"file_size_bytes"`
	DurationSeconds   *float64  `json:"duration_seconds,omitempty"`
	Width             *int      `json:"width,omitempty"`
	Height            *int      `json:"height,omitempty"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
