package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/constants"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/helpers"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/models"
)

func ProcessVideo(
	inputPath string,
	outputFolder string,
) error {

	outputBaseDir := filepath.Join(
		constants.HLSDir,
		outputFolder,
	)

	if err := helpers.EnsureDirectory(outputBaseDir); err != nil {
		return err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	successful := make([]models.Resolution, 0)

	for _, q := range models.Qualities {

		wg.Add(1)

		go func(res models.Resolution) {

			defer wg.Done()

			resDir := filepath.Join(
				outputBaseDir,
				res.Name,
			)

			if err := helpers.EnsureDirectory(resDir); err != nil {
				log.Printf("Directory error %s: %v", res.Name, err)
				return
			}

			segmentPath := filepath.ToSlash(
				filepath.Join(resDir, "segment_%03d.ts"),
			)

			playlistPath := filepath.ToSlash(
				filepath.Join(resDir, "index.m3u8"),
			)

			cmd := exec.Command(
				"ffmpeg",
				"-y",
				"-i", inputPath,

				"-vf",
				"scale=w="+res.Width+
					":h="+res.Height+
					":force_original_aspect_ratio=decrease"+
					":force_divisible_by=2",

				"-map", "0:v:0",
				"-c:v", "libx264",
				"-preset", "veryfast",
				"-b:v", res.VideoBit,

				"-map", "0:a:0?",
				"-c:a", "aac",
				"-b:a", res.AudioBit,

				"-f", "hls",
				"-hls_time", constants.SegmentDuration,
				"-hls_playlist_type", "vod",
				"-hls_segment_filename", segmentPath,

				playlistPath,
			)

			output, err := cmd.CombinedOutput()

			if err != nil {
				log.Printf(
					"FFmpeg failed (%s): %v\n%s",
					res.Name,
					err,
					string(output),
				)
				return
			}

			if _, err := os.Stat(filepath.FromSlash(playlistPath)); err != nil {
				log.Printf("Playlist missing (%s): %v", res.Name, err)
				return
			}

			mu.Lock()
			successful = append(successful, res)
			mu.Unlock()

			log.Printf("Successfully transcoded: %s", res.Name)

		}(q)
	}

	wg.Wait()

	if len(successful) == 0 {
		return fmt.Errorf("all transcoding jobs failed")
	}

	// Preserve quality order
	sort.Slice(successful, func(i, j int) bool {
		for _, q := range models.Qualities {
			if successful[i].Name == q.Name {
				return true
			}
			if successful[j].Name == q.Name {
				return false
			}
		}
		return false
	})

	if err := helpers.GenerateMasterPlaylist(
		outputBaseDir,
		successful,
	); err != nil {
		return fmt.Errorf("master playlist: %w", err)
	}

	log.Printf("Processing completed: %s", outputFolder)

	return nil
}
