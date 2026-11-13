
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv            string
	HTTPAddr          string
	DatabaseURL       string
	DBMaxConns        int32
	DBMinConns        int32

	MediaMTXAPI       string
	MediaMTXRTSP      string

	FFmpegPath        string
	HLSDir            string

	StreamGracePeriod time.Duration
	LeaseTTL          time.Duration
	LeaseHeartbeat    time.Duration

	FFmpegReadyTimeout time.Duration
	FFmpegRestartLimit int
}

func Load() (*Config, error) {

	// Load .env if available.
	// Existing OS environment variables take precedence.
	_ = godotenv.Load()

	maxConns, err := strconv.ParseInt(
		getEnv("DB_MAX_CONNS", "10"), 10, 32,
	)
	if err != nil || maxConns < 1 {
		return nil, fmt.Errorf(
			"DB_MAX_CONNS must be a positive integer",
		)
	}

	minConns, err := strconv.ParseInt(
		getEnv("DB_MIN_CONNS", "2"), 10, 32,
	)
	if err != nil || minConns < 0 || minConns > maxConns {
		return nil, fmt.Errorf(
			"DB_MIN_CONNS must be between 0 and DB_MAX_CONNS",
		)
	}

	streamGracePeriod, err := parseDuration(
		"STREAM_GRACE_PERIOD", "30s",
	)
	if err != nil || streamGracePeriod < 0 {
		return nil, fmt.Errorf(
			"STREAM_GRACE_PERIOD must be a non-negative duration",
		)
	}

	leaseTTL, err := parseDuration(
		"LEASE_TTL", "45s",
	)
	if err != nil || leaseTTL <= 0 {
		return nil, fmt.Errorf(
			"LEASE_TTL must be a positive duration",
		)
	}

	leaseHeartbeat, err := parseDuration(
		"LEASE_HEARTBEAT", "15s",
	)
	if err != nil || leaseHeartbeat <= 0 {
		return nil, fmt.Errorf(
			"LEASE_HEARTBEAT must be a positive duration",
		)
	}

	if leaseHeartbeat >= leaseTTL {
		return nil, fmt.Errorf(
			"LEASE_HEARTBEAT must be less than LEASE_TTL",
		)
	}

	ffmpegReadyTimeout, err := parseDuration(
		"FFMPEG_READY_TIMEOUT", "30s",
	)
	if err != nil || ffmpegReadyTimeout <= 0 {
		return nil, fmt.Errorf(
			"FFMPEG_READY_TIMEOUT must be a positive duration",
		)
	}

	restartLimit, err := strconv.Atoi(
		getEnv("FFMPEG_RESTART_LIMIT", "5"),
	)
	if err != nil || restartLimit < 0 {
		return nil, fmt.Errorf(
			"FFMPEG_RESTART_LIMIT must be a non-negative integer",
		)
	}

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),

		DBMaxConns: int32(maxConns),
		DBMinConns: int32(minConns),

		MediaMTXAPI:  getEnv("MEDIAMTX_API", "http://127.0.0.1:9997"),
		MediaMTXRTSP: getEnv("MEDIAMTX_RTSP", "rtsp://127.0.0.1:8554"),

		FFmpegPath: getEnv("FFMPEG_PATH", "ffmpeg"),
		HLSDir:     getEnv("HLS_DIR", "./hls"),

		StreamGracePeriod:  streamGracePeriod,
		LeaseTTL:           leaseTTL,
		LeaseHeartbeat:     leaseHeartbeat,
		FFmpegReadyTimeout: ffmpegReadyTimeout,
		FFmpegRestartLimit: restartLimit,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func parseDuration(key, fallback string) (time.Duration, error) {
	return time.ParseDuration(getEnv(key, fallback))
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
