
package models

import "time"

type VideoVariant struct {
	ID         string    `json:"id"`
	VideoID    string    `json:"video_id"`
	Quality    string    `json:"quality"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	Bitrate    *int      `json:"bitrate,omitempty"`
	PlaylistURL *string  `json:"playlist_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
