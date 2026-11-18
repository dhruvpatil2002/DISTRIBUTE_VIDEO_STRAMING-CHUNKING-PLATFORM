
package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type MediaMTXManager struct {
	APIURL     string
	RTSPURL    string
	 APIUser    string
    APIPass    string
    HTTPClient *http.Client
	
}

func NewMediaMTXManager(
	apiURL, rtspURL, apiUser, apiPass string,
) *MediaMTXManager {
	return &MediaMTXManager{
		APIURL:  strings.TrimRight(apiURL, "/"),
		RTSPURL: strings.TrimRight(rtspURL, "/"),
		APIUser: apiUser,
		APIPass: apiPass,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// cameraPath generates a stable MediaMTX path.
func cameraPath(cameraID string) string {
	return "camera_" + strings.ReplaceAll(cameraID, "-", "")
}

// validateCameraID prevents invalid path names.
func validateCameraID(cameraID string) error {
	if strings.TrimSpace(cameraID) == "" {
		return fmt.Errorf("camera ID is required")
	}

	for _, char := range cameraID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return fmt.Errorf("invalid camera ID")
		}
	}

	return nil
}

// validateSourceURL validates the camera's RTSP source.
func validateSourceURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid RTSP source URL")
	}

	if parsed.Scheme != "rtsp" &&
		parsed.Scheme != "rtsps" {
		return fmt.Errorf("RTSP or RTSPS URL required")
	}

	if parsed.Hostname() == "" {
		return fmt.Errorf("RTSP hostname is required")
	}

	return nil
}

// RegisterCamera adds a persistent RTSP source to MediaMTX.
// sourceOnDemand=false keeps the source connected continuously.
func (m *MediaMTXManager) RegisterCamera(
	ctx context.Context,
	cameraID string,
	rtspURL string,
) error {

	if err := validateCameraID(cameraID); err != nil {
		return err
	}

	if err := validateSourceURL(rtspURL); err != nil {
		return err
	}

	payload := map[string]any{
		"source":         rtspURL,
		"sourceOnDemand": false,
		"rtspTransport":  "tcp",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf(
		"%s/v3/config/paths/add/%s",
		m.APIURL,
		url.PathEscape(cameraPath(cameraID)),
	)

	return m.doRequest(
		ctx,
		http.MethodPost,
		endpoint,
		body,
	)
}

// UpdateCamera replaces an existing camera source.
func (m *MediaMTXManager) UpdateCamera(
	ctx context.Context,
	cameraID string,
	rtspURL string,
) error {

	if err := validateCameraID(cameraID); err != nil {
		return err
	}

	if err := validateSourceURL(rtspURL); err != nil {
		return err
	}

	payload := map[string]any{
		"source":         rtspURL,
		"sourceOnDemand": false,
		"rtspTransport":  "tcp",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf(
		"%s/v3/config/paths/replace/%s",
		m.APIURL,
		url.PathEscape(cameraPath(cameraID)),
	)

	return m.doRequest(
		ctx,
		http.MethodPost,
		endpoint,
		body,
	)
}

func (m *MediaMTXManager) RemoveCamera(
    ctx context.Context,
    cameraID string,
) error {
    if err := validateCameraID(cameraID); err != nil {
        return err
    }

    endpoint := fmt.Sprintf(
        "%s/v3/config/paths/delete/%s",
        m.APIURL,
        url.PathEscape(cameraPath(cameraID)),
    )

    req, err := http.NewRequestWithContext(
        ctx,
        http.MethodPost,
        endpoint,
        nil,
    )
    if err != nil {
        return fmt.Errorf("create MediaMTX delete request: %w", err)
    }

    req.SetBasicAuth(m.APIUser, m.APIPass)

    client := m.HTTPClient
    if client == nil {
        client = &http.Client{Timeout: 5 * time.Second}
    }

    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("MediaMTX delete request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusNotFound {
        // Treat an already-absent relay as successful cleanup.
        // This assumes the API endpoint itself is valid.
        return nil
    }

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

        return fmt.Errorf(
            "MediaMTX delete failed: status=%d response=%s",
            resp.StatusCode,
            strings.TrimSpace(string(body)),
        )
    }

    return nil
}
// GetCameraRTSPURL returns the internal relay URL.
// FFmpeg reads from MediaMTX, not directly from the camera.
func (m *MediaMTXManager) GetCameraRTSPURL(
	cameraID string,
) (string, error) {

	if err := validateCameraID(cameraID); err != nil {
		return "", err
	}

	base, err := url.Parse(m.RTSPURL)
	if err != nil {
		return "", fmt.Errorf("invalid MediaMTX RTSP address")
	}

	if base.Scheme != "rtsp" && base.Scheme != "rtsps" {
		return "", fmt.Errorf("invalid MediaMTX RTSP scheme")
	}

	if base.Hostname() == "" {
		return "", fmt.Errorf("MediaMTX RTSP hostname is required")
	}

	base.Path = strings.TrimRight(base.Path, "/") +
		"/" + cameraPath(cameraID)

	return base.String(), nil
}

// doRequest performs MediaMTX API requests.
func (m *MediaMTXManager) doRequest(
	ctx context.Context,
	method string,
	endpoint string,
	body []byte,
) error {

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.SetBasicAuth(m.APIUser, m.APIPass)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := m.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("MediaMTX API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(
			io.LimitReader(resp.Body, 4096),
		)

		return fmt.Errorf(
			"MediaMTX API returned status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return nil
}

func (m *MediaMTXManager) EnsureCamera(
    ctx context.Context,
    cameraID string,
    rtspURL string,
) error {
    if err := validateCameraID(cameraID); err != nil {
        return err
    }

    if err := validateSourceURL(rtspURL); err != nil {
        return err
    }

    path := url.PathEscape(cameraPath(cameraID))

    // Check whether the camera path exists.
    getURL := fmt.Sprintf(
        "%s/v3/config/paths/get/%s",
        m.APIURL,
        path,
    )

    req, err := http.NewRequestWithContext(
        ctx,
        http.MethodGet,
        getURL,
        nil,
    )
    if err != nil {
        return err
    }

    req.SetBasicAuth(m.APIUser, m.APIPass)

    client := m.HTTPClient
    if client == nil {
        client = &http.Client{Timeout: 5 * time.Second}
    }

    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("check MediaMTX path: %w", err)
    }

    // Close the response before sending another request.
    io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
    resp.Body.Close()

    switch resp.StatusCode {
    case http.StatusOK:
        // Already exists: synchronize its configuration.
        return m.UpdateCamera(ctx, cameraID, rtspURL)

    case http.StatusNotFound:
        // Missing: create the path.
        return m.RegisterCamera(ctx, cameraID, rtspURL)

    default:
        return fmt.Errorf(
            "check MediaMTX path failed: HTTP %d",
            resp.StatusCode,
        )
    }
}