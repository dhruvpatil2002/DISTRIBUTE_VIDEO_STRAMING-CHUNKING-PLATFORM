package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/constants"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/helpers"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/service"
)

func UploadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed",
			http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		constants.MaxUploadSize,
	)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Invalid upload or file too large",
			http.StatusBadRequest)
		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile(constants.VideoFormKey)
	if err != nil {
		http.Error(w, "Video file is required",
			http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := filepath.Ext(filepath.Base(header.Filename))
	if ext == "" {
		ext = ".mp4"
	}

	inputPath, err := helpers.SaveUploadedFile(
		file,
		constants.UploadDir,
		ext,
	)
	if err != nil {
		log.Printf("Upload save error: %v", err)
		http.Error(w, "Failed to save video",
			http.StatusInternalServerError)
		return
	}

	outputFolder := strings.TrimSuffix(
		filepath.Base(inputPath),
		ext,
	)

	// Respond before processing begins.
	w.WriteHeader(http.StatusAccepted)
	_, _ = fmt.Fprintln(w, "Upload successful. Transcoding started.")

	go func() {

		defer func() {
			if err := os.Remove(inputPath); err != nil {
				log.Printf("Original file cleanup error: %v", err)
			}
		}()

		if err := service.ProcessVideo(
			inputPath,
			outputFolder,
		); err != nil {
			log.Printf("Video processing failed: %v", err)
		}
	}()
}
