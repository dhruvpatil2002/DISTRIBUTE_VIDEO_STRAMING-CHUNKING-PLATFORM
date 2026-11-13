
package models

import "time"

type TranscodingJob struct {
	ID           string     `json:"id"`
	VideoID      string     `json:"video_id"`
	JobType      string     `json:"job_type"`
	Status       string     `json:"status"`
	Progress     int        `json:"progress"`
	Attempts     int        `json:"attempts"`
	WorkerID     *string    `json:"worker_id,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
