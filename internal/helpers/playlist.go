package helpers

import (
	"fmt"
	"os"
	"path/filepath"

	"distribute_video_streaming_chunking_platform/internal/models"
)

func GenerateMasterPlaylist(
	outputDir string,
	successful []models.Resolution,
) error {

	if len(successful) == 0 {
		return fmt.Errorf("no successful resolutions")
	}

	if err := EnsureDirectory(outputDir); err != nil {
		return err
	}

	masterPath := filepath.Join(
		outputDir,
		"master.m3u8",
	)

	file, err := os.Create(masterPath)
	if err != nil {
		return fmt.Errorf("create master playlist: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(
		"#EXTM3U\n#EXT-X-VERSION:3\n",
	); err != nil {
		return err
	}

	for _, q := range successful {

		entry := fmt.Sprintf(
			"#EXT-X-STREAM-INF:BANDWIDTH=%s,RESOLUTION=%sx%s\n%s/index.m3u8\n",
			q.Bandwidth,
			q.Width,
			q.Height,
			q.Name,
		)

		if _, err := file.WriteString(entry); err != nil {
			return err
		}
	}

	if err := file.Sync(); err != nil {
		return err
	}

	return nil
}
