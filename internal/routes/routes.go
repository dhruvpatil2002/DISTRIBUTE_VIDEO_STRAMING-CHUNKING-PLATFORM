package routes

import (
	"net/http"

	"distribute_video_streaming_chunking_platform/internal/handler"
	"distribute_video_streaming_chunking_platform/internal/middleware"
)

func RegisterRoutes() {

	http.HandleFunc(
		"/upload",
		handlers.UploadHandler,
	)

	http.Handle(
		"/hls/",
		middleware.CORS(
			handlers.StreamHandler(),
		),
	)
}
