package models

import "time"

type LiveStream struct {
    ID          string     `json:"id"`
    UserID      *string    `json:"user_id,omitempty"`
    Name        string     `json:"name"`
    RTSPURL     string     `json:"-"`
    Status      string     `json:"status"`
    CurrentQuality string  `json:"current_quality"`
    HLSURL      *string    `json:"hls_url,omitempty"`
    ErrorMessage *string   `json:"error_message,omitempty"`
    StartedAt   *time.Time `json:"started_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}