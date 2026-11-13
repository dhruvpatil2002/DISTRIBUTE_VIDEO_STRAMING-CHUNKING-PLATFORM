package models

type CameraVariant struct {
    ID          string `json:"id"`
    CameraID    string `json:"camera_id"`
    Quality     string `json:"quality"`
    Width       int    `json:"width"`
    Height      int    `json:"height"`
    PlaylistURL string `json:"playlist_url"`
}