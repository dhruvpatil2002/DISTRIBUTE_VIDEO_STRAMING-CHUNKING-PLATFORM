package handlers

import (
	"net/http"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/constants"
)

func StreamHandler() http.Handler {

	return http.StripPrefix(
		"/hls/",
		http.FileServer(
			http.Dir(constants.HLSDir),
		),
	)
}
