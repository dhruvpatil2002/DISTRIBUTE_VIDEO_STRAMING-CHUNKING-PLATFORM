package handlers

import (
	"net/http"

	"distribute_video_streaming_chunking_platform/internal/constants"
)

func StreamHandler() http.Handler {

	return http.StripPrefix(
		"/hls/",
		http.FileServer(
			http.Dir(constants.HLSDir),
		),
	)
}
