
package stream

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrLeaseExpired   = errors.New("playback lease expired")
	ErrLeaseForbidden = errors.New("playback lease forbidden")
	ErrStreamFailed   = errors.New("stream failed")
	ErrManagerClosed  = errors.New("stream manager closed")
)

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
	Ctx    context.Context
	Cancel context.CancelFunc

	Viewers map[string]struct{}
	Dir     string

	Status    StreamStatus
	LastError string

	Done    chan struct{}
	Changed chan struct{}

	StopTimer *time.Timer
}

type HLSManager struct {
	mu sync.Mutex

	processes map[string]*CameraProcess
	leases    map[string]*ViewerLease

	ffmpegPath   string
	hlsDir       string
	gracePeriod  time.Duration
	leaseTTL     time.Duration
	readyTimeout time.Duration
	maxRestarts  int

	ctx    context.Context
	cancel context.CancelFunc

	wg sync.WaitGroup
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

	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

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

	ctx, cancel := context.WithCancel(context.Background())

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

// Must be called while holding m.mu.
func (m *HLSManager) notifyLocked(p *CameraProcess) {
	close(p.Changed)
	p.Changed = make(chan struct{})
}

func validCameraID(id string) bool {
	if len(id) != 36 {
		return false
	}

	for i, ch := range id {
		switch i {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
		default:
			if !((ch >= '0' && ch <= '9') ||
				(ch >= 'a' && ch <= 'f') ||
				(ch >= 'A' && ch <= 'F')) {
				return false
			}
		}
	}

	return true
}

func (m *HLSManager) Acquire(
	cameraID, rtspURL, userID string,
) (string, string, error) {

	if !validCameraID(cameraID) {
		return "", "", errors.New("invalid camera UUID")
	}

	if rtspURL == "" || userID == "" {
		return "", "", errors.New("missing playback parameters")
	}

	leaseID, err := generateLeaseID()
	if err != nil {
		return "", "", err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ctx.Err() != nil {
		return "", "", ErrManagerClosed
	}

	p := m.processes[cameraID]

	if p != nil && p.Status == StreamFailed {
		delete(m.processes, cameraID)
		p.Cancel()
		p = nil
	}

	if p == nil {
		dir := filepath.Join(m.hlsDir, cameraID)

		if err := os.MkdirAll(dir, 0750); err != nil {
			return "", "", fmt.Errorf("create HLS directory: %w", err)
		}

		ctx, cancel := context.WithCancel(m.ctx)

		p = &CameraProcess{
			Ctx:     ctx,
			Cancel:  cancel,
			Viewers: make(map[string]struct{}),
			Dir:     dir,
			Status:  StreamStarting,
			Done:    make(chan struct{}),
			Changed: make(chan struct{}),
		}

		m.processes[cameraID] = p

		m.wg.Add(1)
		go m.supervise(cameraID, rtspURL, p)

		log.Printf("[HLS] Created camera process camera=%s", cameraID)
	}

	if p.StopTimer != nil {
		p.StopTimer.Stop()
		p.StopTimer = nil
	}

	now := time.Now()

	p.Viewers[leaseID] = struct{}{}

	m.leases[leaseID] = &ViewerLease{
		ID:        leaseID,
		CameraID:  cameraID,
		UserID:    userID,
		LastSeen:  now,
		ExpiresAt: now.Add(m.leaseTTL),
	}

	log.Printf(
		"[HLS] Viewer acquired camera=%s viewers=%d",
		cameraID,
		len(p.Viewers),
	)

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

	ctx, cancel := context.WithTimeout(ctx, m.readyTimeout)
	defer cancel()

	for {
		m.mu.Lock()

		p := m.processes[cameraID]
		if p == nil {
			m.mu.Unlock()
			return ErrStreamFailed
		}

		status := p.Status
		lastError := p.LastError
		changed := p.Changed
		done := p.Done

		m.mu.Unlock()

		switch status {
		case StreamReady:
			return nil

		case StreamFailed:
			return fmt.Errorf(
				"%w: %s",
				ErrStreamFailed,
				lastError,
			)
		}

		select {
		case <-changed:
			continue

		case <-done:
			return ErrStreamFailed

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *HLSManager) supervise(
	cameraID, rtspURL string,
	p *CameraProcess,
) {
	defer m.wg.Done()
	defer close(p.Done)

	for attempt := 0; attempt <= m.maxRestarts; attempt++ {
		if p.Ctx.Err() != nil {
			return
		}

		m.mu.Lock()

		if m.processes[cameraID] != p {
			m.mu.Unlock()
			return
		}

		if attempt == 0 {
			p.Status = StreamStarting
		} else {
			p.Status = StreamRecovering
		}

		m.notifyLocked(p)
		m.mu.Unlock()

		log.Printf(
			"[HLS] Starting camera=%s attempt=%d",
			cameraID,
			attempt+1,
		)

		err := m.startFFmpeg(cameraID, p, rtspURL)

		if p.Ctx.Err() != nil {
			return
		}

		m.mu.Lock()

		if m.processes[cameraID] != p {
			m.mu.Unlock()
			return
		}

		if err != nil {
			p.LastError = err.Error()
		}

		p.Status = StreamRecovering
		m.notifyLocked(p)

		m.mu.Unlock()

		log.Printf(
			"[HLS] Camera=%s FFmpeg stopped: %v",
			cameraID,
			err,
		)

		if attempt == m.maxRestarts {
			m.mu.Lock()

			if m.processes[cameraID] == p {
				p.Status = StreamFailed
				m.notifyLocked(p)
			}

			m.mu.Unlock()

			log.Printf("[HLS] Camera=%s failed permanently", cameraID)
			return
		}

		delay := time.Second * time.Duration(1<<min(attempt, 5))

		timer := time.NewTimer(delay)

		select {
		case <-p.Ctx.Done():
			timer.Stop()
			return

		case <-timer.C:
		}
	}
}

func (m *HLSManager) startFFmpeg(
	cameraID string,
	p *CameraProcess,
	rtspURL string,
) error {

	ctx, cancel := context.WithCancel(p.Ctx)
	defer cancel()

	// Remove old HLS output before starting a new generation.
	// This directory must be dedicated to this camera.
	if err := os.RemoveAll(p.Dir); err != nil {
		return fmt.Errorf("clean old HLS files: %w", err)
	}

	if err := os.MkdirAll(p.Dir, 0750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	for _, variant := range []string{"0", "1", "2"} {
		dir := filepath.Join(p.Dir, variant)

		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf(
				"create variant directory %s: %w",
				dir,
				err,
			)
		}
	}

	args := buildFFmpegArgs(rtspURL, p.Dir)

	cmd := exec.CommandContext(
		ctx,
		m.ffmpegPath,
		args...,
	)

	// Never print RTSP credentials or the full command.
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	log.Printf("[HLS] Starting FFmpeg camera=%s", cameraID)
	log.Printf("[HLS] Output directory=%s", p.Dir)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start FFmpeg: %w", err)
	}

	log.Printf(
		"[HLS] FFmpeg started camera=%s PID=%d",
		cameraID,
		cmd.Process.Pid,
	)

	waitResult := make(chan error, 1)

	go func() {
		waitResult <- cmd.Wait()
	}()

	readyCtx, readyCancel := context.WithTimeout(
		ctx,
		m.readyTimeout,
	)
	defer readyCancel()

	readyResult := make(chan error, 1)

	go func() {
		readyResult <- waitForHLS(readyCtx, p.Dir)
	}()

	select {
	case err := <-readyResult:
		if err != nil {
			cancel()
			<-waitResult
			return fmt.Errorf("HLS readiness failed: %w", err)
		}

	case err := <-waitResult:
		cancel()

		if err == nil {
			return errors.New("FFmpeg exited before HLS readiness")
		}

		return fmt.Errorf("FFmpeg exited: %w", err)

	case <-readyCtx.Done():
		cancel()
		<-waitResult
		return fmt.Errorf("HLS readiness timeout: %w", readyCtx.Err())
	}

	// Check whether FFmpeg has already exited.
	select {
	case err := <-waitResult:
		if err == nil {
			return errors.New("FFmpeg exited after HLS readiness")
		}
		return fmt.Errorf("FFmpeg exited: %w", err)
	default:
	}

	m.mu.Lock()

	if p.Ctx.Err() == nil && m.processes[cameraID] == p {
		p.Status = StreamReady
		p.LastError = ""
		m.notifyLocked(p)
	}

	m.mu.Unlock()

	log.Printf("[HLS] Camera=%s stream READY", cameraID)

	select {
	case err := <-waitResult:
		if err != nil {
			return fmt.Errorf("FFmpeg exited: %w", err)
		}

		return errors.New("FFmpeg exited unexpectedly")

	case <-p.Ctx.Done():
		cancel()
		<-waitResult
		return p.Ctx.Err()
	}
}

func buildFFmpegArgs(rtspURL, outputDir string) []string {
	return []string{
		"-hide_banner",
		"-loglevel", "warning",
		"-nostdin",

		"-rtsp_transport", "tcp",
		"-i", rtspURL,

		"-map", "0:v:0",
		"-map", "0:v:0",
		"-map", "0:v:0",

		"-c:v", "libx264",
		"-preset", "veryfast",
		"-pix_fmt", "yuv420p",

		"-filter:v:0",
		"scale=w=1920:h=1080:force_original_aspect_ratio=decrease",
		"-b:v:0", "5000k",
		"-maxrate:v:0", "5350k",
		"-bufsize:v:0", "7500k",

		"-filter:v:1",
		"scale=w=1280:h=720:force_original_aspect_ratio=decrease",
		"-b:v:1", "2800k",
		"-maxrate:v:1", "2996k",
		"-bufsize:v:1", "4200k",

		"-filter:v:2",
		"scale=w=854:h=480:force_original_aspect_ratio=decrease",
		"-b:v:2", "1400k",
		"-maxrate:v:2", "1498k",
		"-bufsize:v:2", "2100k",

		"-r", "30",
		"-g", "60",
		"-keyint_min", "60",
		"-sc_threshold", "0",

		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "6",
		"-hls_flags", "delete_segments+independent_segments",
		"-hls_segment_type", "mpegts",

		"-var_stream_map", "v:0 v:1 v:2",
		"-master_pl_name", "master.m3u8",

		"-hls_segment_filename",
		filepath.Join(outputDir, "%v", "segment_%06d.ts"),

		filepath.Join(outputDir, "%v", "index.m3u8"),
	}
}

// waitForHLS waits for a master playlist and a real video segment.
// It must honor context cancellation.
func waitForHLS(
	ctx context.Context,
	outputDir string,
) error {

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := checkHLSReady(outputDir); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"waiting for HLS files: %w",
				ctx.Err(),
			)

		case <-ticker.C:
		}
	}
}

func checkHLSReady(outputDir string) error {
	masterPath := filepath.Join(outputDir, "master.m3u8")

	master, err := os.ReadFile(masterPath)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(string(master), "#EXTM3U") {
		return errors.New("invalid master playlist")
	}

	for _, variant := range []string{"0", "1", "2"} {
		variantDir := filepath.Join(outputDir, variant)
		playlistPath := filepath.Join(variantDir, "index.m3u8")

		playlist, err := os.Open(playlistPath)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(playlist)

		var segmentName string

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			if strings.HasSuffix(line, ".ts") {
				segmentName = line
				break
			}
		}

		scanErr := scanner.Err()
		_ = playlist.Close()

		if scanErr != nil || segmentName == "" {
			continue
		}

		// Only accept the segment filename, not arbitrary paths.
		if filepath.Base(segmentName) != segmentName {
			continue
		}

		segmentPath := filepath.Join(variantDir, segmentName)

		info, err := os.Stat(segmentPath)
		if err == nil && info.Size() > 0 {
			return nil
		}
	}

	return errors.New("no playable HLS segment found")
}

func (m *HLSManager) Heartbeat(
	leaseID, userID string,
) error {

	m.mu.Lock()
	defer m.mu.Unlock()

	lease, ok := m.leases[leaseID]
	if !ok {
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

func (m *HLSManager) Release(
	leaseID, userID string,
) error {

	m.mu.Lock()
	defer m.mu.Unlock()

	lease, ok := m.leases[leaseID]
	if !ok {
		return ErrLeaseExpired
	}

	if lease.UserID != userID {
		return ErrLeaseForbidden
	}

	if !time.Now().Before(lease.ExpiresAt) {
		m.releaseLocked(lease)
		return ErrLeaseExpired
	}

	m.releaseLocked(lease)
	return nil
}

// Must be called while holding m.mu.
func (m *HLSManager) releaseLocked(lease *ViewerLease) {
	delete(m.leases, lease.ID)

	p := m.processes[lease.CameraID]
	if p == nil {
		return
	}

	delete(p.Viewers, lease.ID)

	if len(p.Viewers) > 0 {
		return
	}

	if p.StopTimer != nil {
		p.StopTimer.Stop()
	}

	cameraID := lease.CameraID

	p.StopTimer = time.AfterFunc(m.gracePeriod, func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		if m.processes[cameraID] != p {
			return
		}

		if len(p.Viewers) != 0 {
			return
		}

		log.Printf("[HLS] Stopping idle camera=%s", cameraID)

		delete(m.processes, cameraID)
		p.Cancel()
	})
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

// StopCamera stops the camera process and invalidates its leases.
// Call this when deleting or disabling a camera.
func (m *HLSManager) StopCamera(cameraID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := m.processes[cameraID]
	if p == nil {
		return
	}

	if p.StopTimer != nil {
		p.StopTimer.Stop()
	}

	for leaseID := range p.Viewers {
		delete(m.leases, leaseID)
	}

	delete(m.processes, cameraID)
	p.Cancel()

	log.Printf("[HLS] Camera stopped camera=%s", cameraID)
}

func (m *HLSManager) Shutdown() {
	m.cancel()

	m.mu.Lock()

	for _, p := range m.processes {
		if p.StopTimer != nil {
			p.StopTimer.Stop()
		}

		p.Cancel()
	}

	m.mu.Unlock()

	m.wg.Wait()

	log.Println("[HLS] Manager shutdown complete")
}
