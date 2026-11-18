package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string
	DBMaxConns  int32
	DBMinConns  int32
	JWTSecret   string

	MediaMTXAPI         string
	MediaMTXRTSP        string
	MediaMTXAPIUser     string
	MediaMTXAPIPassword string
	FFmpegPath          string
	HLSDir              string

	StreamGracePeriod time.Duration
	LeaseTTL          time.Duration
	LeaseHeartbeat    time.Duration

	FFmpegReadyTimeout time.Duration
	FFmpegRestartLimit int
}

func Load() (*Config, error) {

	_ = godotenv.Load()

	maxConns, err := strconv.ParseInt(
		getEnv("DB_MAX_CONNS", "10"), 10, 32,
	)
	if err != nil || maxConns < 1 {
		return nil, fmt.Errorf("invalid DB_MAX_CONNS")
	}

	minConns, err := strconv.ParseInt(
		getEnv("DB_MIN_CONNS", "2"), 10, 32,
	)
	if err != nil || minConns < 0 || minConns > maxConns {
		return nil, fmt.Errorf("invalid DB_MIN_CONNS")
	}

	streamGracePeriod, err := parseDuration(
		"STREAM_GRACE_PERIOD", "30s",
	)
	if err != nil || streamGracePeriod < 0 {
		return nil, fmt.Errorf("invalid STREAM_GRACE_PERIOD")
	}

	leaseTTL, err := parseDuration("LEASE_TTL", "45s")
	if err != nil || leaseTTL <= 0 {
		return nil, fmt.Errorf("invalid LEASE_TTL")
	}

	leaseHeartbeat, err := parseDuration(
		"LEASE_HEARTBEAT", "15s",
	)
	if err != nil || leaseHeartbeat <= 0 ||
		leaseHeartbeat >= leaseTTL {
		return nil, fmt.Errorf("invalid LEASE_HEARTBEAT")
	}

	ffmpegReadyTimeout, err := parseDuration(
		"FFMPEG_READY_TIMEOUT", "30s",
	)
	if err != nil || ffmpegReadyTimeout <= 0 {
		return nil, fmt.Errorf("invalid FFMPEG_READY_TIMEOUT")
	}

	restartLimit, err := strconv.Atoi(
		getEnv("FFMPEG_RESTART_LIMIT", "5"),
	)
	if err != nil || restartLimit < 0 {
		return nil, fmt.Errorf("invalid FFMPEG_RESTART_LIMIT")
	}

	jwtSecret := getEnv("JWT_SECRET", "")

	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf(
			"JWT_SECRET must contain at least 32 characters",
		)
	}
	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),

		DBMaxConns: int32(maxConns),
		DBMinConns: int32(minConns),

		JWTSecret: jwtSecret,

		MediaMTXAPI:         getEnv("MEDIAMTX_API", "http://127.0.0.1:9997"),
		MediaMTXRTSP:        getEnv("MEDIAMTX_RTSP", "rtsp://127.0.0.1:8554"),
		MediaMTXAPIUser:     os.Getenv("MEDIAMTX_API_USER"),
		MediaMTXAPIPassword: os.Getenv("MEDIAMTX_API_PASSWORD"),

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
	if cfg.MediaMTXAPIUser == "" || cfg.MediaMTXAPIPassword == "" {
		return nil, fmt.Errorf("MediaMTX API credentials are required")
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
