package stream

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

func WaitForHLS(
    ctx context.Context,
    outputDir string,
) error {

    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    for {

        select {

        case <-ctx.Done():
            return ctx.Err()

        case <-ticker.C:

            master := filepath.Join(
                outputDir,
                "master.m3u8",
            )

            info, err := os.Stat(master)

            if err != nil || info.Size() == 0 {
                continue
            }

            // Ensure all expected quality playlists exist.
            ready := true

            for _, quality := range []string{
                "1080p",
                "720p",
                "480p",
            } {

                playlist := filepath.Join(
                    outputDir,
                    quality,
                    "index.m3u8",
                )

                info, err := os.Stat(playlist)

                if err != nil || info.Size() == 0 {
                    ready = false
                    break
                }
            }

            if ready {
                return nil
            }
        }
    }
}

func WaitForHLSWithTimeout(
    ctx context.Context,
    outputDir string,
    timeout time.Duration,
) error {

    readyCtx, cancel := context.WithTimeout(
        ctx,
        timeout,
    )
    defer cancel()

    if err := WaitForHLS(readyCtx, outputDir); err != nil {
        return fmt.Errorf(
            "HLS readiness failed: %w",
            err,
        )
    }

    return nil
}