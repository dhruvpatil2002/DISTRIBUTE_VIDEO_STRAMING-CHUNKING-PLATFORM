
package stream

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrLeaseExpired = errors.New("playback lease expired")
	ErrStreamFailed = errors.New("stream failed")
	ErrLeaseForbidden = errors.New(
    "playback lease does not belong to user",
))

type StreamStatus string

const (
	StreamStarting   StreamStatus = "starting"
	StreamReady      StreamStatus = "ready"
	StreamRecovering StreamStatus = "recovering"
	StreamFailed     StreamStatus = "failed"
)

type ViewerLease struct {
	ID        string
	CameraID  string
	UserID    string
	LastSeen  time.Time
	ExpiresAt time.Time
}

type CameraProcess struct {
	Cmd       *exec.Cmd
	Cancel    context.CancelFunc
	Viewers   map[string]struct{}
	Timer     *time.Timer
	Dir       string
	Status    StreamStatus
	Ready     chan struct{}
	Done      chan struct{}
	ReadyOnce sync.Once
	LastError string
}

type HLSManager struct {
	mu sync.Mutex

	processes map[string]*CameraProcess
	leases    map[string]*ViewerLease

	ffmpegPath string
	hlsDir     string

	gracePeriod time.Duration
	leaseTTL    time.Duration
	readyTimeout time.Duration
	maxRestarts int

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func generateLeaseID() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func NewHLSManager(
	ffmpegPath string,
	hlsDir string,
	gracePeriod time.Duration,
	leaseTTL time.Duration,
	readyTimeout time.Duration,
	maxRestarts int,
) *HLSManager {

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	if gracePeriod <= 0 {
		gracePeriod = 30 * time.Second
	}

	if leaseTTL <= 0 {
		leaseTTL = 45 * time.Second
	}

	if readyTimeout <= 0 {
		readyTimeout = 30 * time.Second
	}

	if maxRestarts < 0 {
		maxRestarts = 0
	}

	m := &HLSManager{
		processes:    make(map[string]*CameraProcess),
		leases:       make(map[string]*ViewerLease),
		ffmpegPath:   ffmpegPath,
		hlsDir:       hlsDir,
		gracePeriod:  gracePeriod,
		leaseTTL:     leaseTTL,
		readyTimeout: readyTimeout,
		maxRestarts:  maxRestarts,
		ctx:          ctx,
		cancel:       cancel,
	}

	m.wg.Add(1)
	go m.expirationLoop()

	return m
}

func (m *HLSManager) Acquire(
	cameraID string,
	rtspURL string,
	 userID string,
) (string, string, error) {

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ctx.Err() != nil {
		return "", "", errors.New("stream manager shutting down")
	}

	leaseID, err := generateLeaseID()
	if err != nil {
		return "", "", err
	}

	process := m.processes[cameraID]

	if process == nil {

		outputDir := filepath.Join(
			m.hlsDir,
			cameraID,
		)

		if err := os.RemoveAll(outputDir); err != nil {
			return "", "", err
		}

		for _, quality := range []string{
			"1080p",
			"720p",
			"480p",
		} {
			if err := os.MkdirAll(
				filepath.Join(outputDir, quality),
				0750,
			); err != nil {
				return "", "", err
			}
		}

		process = &CameraProcess{
			Viewers: make(map[string]struct{}),
			Dir:     outputDir,
			Status:  StreamStarting,
			Ready:   make(chan struct{}),
			Done:    make(chan struct{}),
		}

		m.processes[cameraID] = process

		m.wg.Add(1)
		go m.supervise(cameraID, rtspURL, process)
	}

	if process.Timer != nil {
		process.Timer.Stop()
		process.Timer = nil
	}

	now := time.Now()

	process.Viewers[leaseID] = struct{}{}

	m.leases[leaseID] = &ViewerLease{
		ID:        leaseID,
		CameraID:  cameraID,
		UserID:    userID,
		LastSeen:  now,
		ExpiresAt: now.Add(m.leaseTTL),
	}

	playlistURL := fmt.Sprintf(
		"/hls/streams/%s/master.m3u8",
		cameraID,
	)

	return leaseID, playlistURL, nil
}

func (m *HLSManager) WaitUntilReady(
	ctx context.Context,
	cameraID string,
) error {

	m.mu.Lock()

	process := m.processes[cameraID]

	m.mu.Unlock()

	if process == nil {
		return ErrStreamFailed
	}

	select {
	case <-process.Ready:
		m.mu.Lock()
		status := process.Status
		m.mu.Unlock()

		if status != StreamReady {
			return ErrStreamFailed
		}

		return nil

	case <-process.Done:
		return ErrStreamFailed

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *HLSManager) Heartbeat(
    leaseID string,
    userID string,
) error {

    m.mu.Lock()
    defer m.mu.Unlock()

    lease, exists := m.leases[leaseID]

    if !exists {
        return ErrLeaseExpired
    }

    if lease.UserID != userID {
        return ErrLeaseForbidden
    }

    now := time.Now()

    if !now.Before(lease.ExpiresAt) {
        m.releaseLocked(lease)
        return ErrLeaseExpired
    }

    lease.LastSeen = now
    lease.ExpiresAt = now.Add(m.leaseTTL)

    return nil
}
func (m *HLSManager) Release(leaseID string) {

	m.mu.Lock()
	defer m.mu.Unlock()

	lease, exists := m.leases[leaseID]

	if !exists {
		return
	}

	m.releaseLocked(lease)
}

func (m *HLSManager) releaseLocked(
	lease *ViewerLease,
) {

	delete(m.leases, lease.ID)

	process := m.processes[lease.CameraID]

	if process == nil {
		return
	}

	delete(process.Viewers, lease.ID)

	if len(process.Viewers) != 0 {
		return
	}

	if process.Timer != nil {
		process.Timer.Stop()
	}

	process.Timer = time.AfterFunc(
		m.gracePeriod,
		func() {
			m.stopIfUnused(
				lease.CameraID,
				process,
			)
		},
	)
}

func (m *HLSManager) stopIfUnused(
	cameraID string,
	process *CameraProcess,
) {

	m.mu.Lock()
	defer m.mu.Unlock()

	current := m.processes[cameraID]

	if current != process {
		return
	}

	if len(process.Viewers) != 0 {
		return
	}

	delete(m.processes, cameraID)

	process.Cancel()
}

func (m *HLSManager) expirationLoop() {

	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {

		select {

		case <-m.ctx.Done():
			return

		case now := <-ticker.C:

			m.mu.Lock()

			for _, lease := range m.leases {

				if !now.Before(lease.ExpiresAt) {
					m.releaseLocked(lease)
				}
			}

			m.mu.Unlock()
		}
	}
}

func (m *HLSManager) supervise(
	cameraID string,
	rtspURL string,
	process *CameraProcess,
) {

	defer m.wg.Done()
	defer close(process.Done)

	var restartCount int

	for {

		if m.ctx.Err() != nil {
			return
		}

		m.mu.Lock()

		current := m.processes[cameraID]

		if current != process {
			m.mu.Unlock()
			return
		}

		if len(process.Viewers) == 0 {
			m.mu.Unlock()
			return
		}

		if restartCount > 0 {
			process.Status = StreamRecovering
		} else {
			process.Status = StreamStarting
		}

		m.mu.Unlock()

		if err := m.startFFmpeg(
			process,
			rtspURL,
		); err != nil {

			m.mu.Lock()
			process.LastError = err.Error()
			m.mu.Unlock()
		}

		select {

		case <-process.CancelContext().Done():
			return

		default:
		}

		if restartCount >= m.maxRestarts {
			m.mu.Lock()
			process.Status = StreamFailed
			m.mu.Unlock()
			return
		}

		restartCount++

		delay := time.Second * time.Duration(
			1<<min(restartCount-1, 5),
		)

		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		timer := time.NewTimer(delay)

		select {

		case <-m.ctx.Done():
			timer.Stop()
			return

		case <-timer.C:
		}

		m.mu.Lock()
		hasViewers := len(process.Viewers) > 0
		current = m.processes[cameraID]
		m.mu.Unlock()

		if !hasViewers || current != process {
			return
		}
	}
}

// startFFmpeg owns each individual FFmpeg process.
// Only the supervisor calls this function.

func (m *HLSManager) startFFmpeg(
	process *CameraProcess,
	rtspURL string,
) error {

	processCtx, cancel := context.WithCancel(m.ctx)

	process.Cancel = cancel

	cmd := exec.CommandContext(
		processCtx,
		m.ffmpegPath,
		buildFFmpegArgs(rtspURL, process.Dir)...,
	)

	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	process.Cmd = cmd

	readyCtx, readyCancel := context.WithTimeout(
		processCtx,
		m.readyTimeout,
	)

	readyErr := WaitForHLS(
		readyCtx,
		process.Dir,
	)

	readyCancel()

	if readyErr == nil {
		m.mu.Lock()

		process.Status = StreamReady
		process.LastError = ""

		process.ReadyOnce.Do(func() {
			close(process.Ready)
		})

		m.mu.Unlock()
	}

	waitErr := cmd.Wait()

	cancel()

	if waitErr != nil {
		return fmt.Errorf("ffmpeg exited: %w", waitErr)
	}

	if readyErr != nil {
		return fmt.Errorf("HLS readiness: %w", readyErr)
	}

	return errors.New("ffmpeg exited unexpectedly")
}

// CancelContext returns the current process context.
// A process cancellation is controlled by its supervisor.

func (p *CameraProcess) CancelContext() context.Context {
	// The supervisor uses the manager context for shutdown.
	// Process completion is detected through Cmd.Wait().
	return context.Background()
}

func (m *HLSManager) Shutdown() {

	m.cancel()

	m.mu.Lock()

	for _, process := range m.processes {
		if process.Timer != nil {
			process.Timer.Stop()
		}

		if process.Cancel != nil {
			process.Cancel()
		}
	}

	m.mu.Unlock()

	m.wg.Wait()
}

func buildFFmpegArgs(
	rtspURL string,
	outputDir string,
) []string {

	filter := strings.Join([]string{
		"[0:v]split=3[v1][v2][v3]",
		"[v1]scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2[v1080]",
		"[v2]scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2[v720]",
		"[v3]scale=854:480:force_original_aspect_ratio=decrease,pad=854:480:(ow-iw)/2:(oh-ih)/2[v480]",
	}, ";")

	return []string{
		"-hide_banner",
		"-loglevel", "warning",
		"-rtsp_transport", "tcp",
		"-fflags", "nobuffer",
		"-flags", "low_delay",
		"-i", rtspURL,
		"-filter_complex", filter,
		"-map", "[v1080]",
		"-map", "[v720]",
		"-map", "[v480]",
		"-an",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-tune", "zerolatency",
		"-pix_fmt", "yuv420p",
		"-b:v:0", "5000k",
		"-b:v:1", "2500k",
		"-b:v:2", "1000k",
		"-maxrate:v:0", "5500k",
		"-maxrate:v:1", "2800k",
		"-maxrate:v:2", "1200k",
		"-g", "50",
		"-sc_threshold", "0",
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "6",
		"-hls_flags",
		"delete_segments+independent_segments+omit_endlist",
		"-master_pl_name", "master.m3u8",
		"-var_stream_map",
		"v:0,name:1080p v:1,name:720p v:2,name:480p",
		"-hls_segment_filename",
		filepath.Join(outputDir, "%v", "segment_%06d.ts"),
		filepath.Join(outputDir, "%v", "index.m3u8"),
	}
}
