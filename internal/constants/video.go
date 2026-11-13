package constants


import "time"

const (
	VideoFormKey = "video"

	MaxUploadSize = 500 << 20

	UploadDir = "./uploads"
	HLSDir    = "./hls"

	SegmentDuration = "10"

	VideoStatusUploaded   = "uploaded"
	VideoStatusQueued     = "queued"
	VideoStatusProcessing = "processing"
	VideoStatusCompleted  = "completed"
	VideoStatusFailed     = "failed"

	JobStatusQueued     = "queued"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"

	Quality1080p = "1080p"
	Quality720p  = "720p"
	Quality480p  = "480p"
)



const (
    CameraStatusRegistered   = "registered"
    CameraStatusConnecting   = "connecting"
    CameraStatusConnected    = "connected"
    CameraStatusDisconnected = "disconnected"
    CameraStatusFailed       = "failed"
)

const (
    StreamStatusStopped    = "stopped"
    StreamStatusStarting   = "starting"
    StreamStatusLive       = "live"
    StreamStatusFailed     = "failed"

    StreamQuality1080p = "1080p"
    StreamQuality720p  = "720p"
    StreamQuality480p  = "480p"
)

const (
    initialRestartDelay = 1 * time.Second
    maximumRestartDelay = 30 * time.Second
    maximumRestarts     = 5
)

