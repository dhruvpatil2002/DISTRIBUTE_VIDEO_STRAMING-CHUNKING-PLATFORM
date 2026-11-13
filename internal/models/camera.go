package models

import "time"

type Camera struct {
    ID        string    `json:"id"`
    SiteID    string    `json:"site_id"`
    Name      string    `json:"name"`
    RTSPURL   string    `json:"-"`
    Status    string    `json:"status"`
    HLSURL    *string   `json:"hls_url,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}