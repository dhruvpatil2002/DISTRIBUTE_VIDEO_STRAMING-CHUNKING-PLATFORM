package main

import (
	"log"
	"net/http"
	"path/filepath"
	"time"

	"distribute_video_streaming_chunking_platform/internal/constants"
	"distribute_video_streaming_chunking_platform/internal/helpers"
	"distribute_video_streaming_chunking_platform/internal/routes"
)

func main() {

	// Resolve absolute directories
	uploadPath, err := filepath.Abs(constants.UploadDir)
	if err != nil {
		log.Fatalf("Upload path error: %v", err)
	}

	hlsPath, err := filepath.Abs(constants.HLSDir)
	if err != nil {
		log.Fatalf("HLS path error: %v", err)
	}

	// Initialize directories
	if err := helpers.EnsureDirectory(uploadPath); err != nil {
		log.Fatalf("Upload directory error: %v", err)
	}

	if err := helpers.EnsureDirectory(hlsPath); err != nil {
		log.Fatalf("HLS directory error: %v", err)
	}

	// Register routes
	routes.RegisterRoutes()

	// Start server
	server := &http.Server{
		Addr:              constants.ServerAddress,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Println("================================")
	log.Println("HLS Video Server Started")
	log.Println("Upload: http://localhost:8080/upload")
	log.Println("Stream: http://localhost:8080/hls/")
	log.Println("================================")

	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
