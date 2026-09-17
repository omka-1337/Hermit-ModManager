package thunderstore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// Profile codes are Thunderstore's legacy profile storage used by r2modman:
// the .r2z archive base64-encoded with a "#r2modman" prefix.
const (
	profileDataPrefix = "#r2modman"
	// MaxProfileCodeSize is the largest archive r2modman shares as a code.
	MaxProfileCodeSize = 20_000_000
)

var (
	profileCode        = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	ErrInvalidCode     = errors.New("invalid profile code")
	ErrProfileTooLarge = errors.New("profile is too large to share as a code")
)

// ValidProfileCode reports whether s looks like a profile code.
func ValidProfileCode(s string) bool { return profileCode.MatchString(s) }

// UploadProfile stores an .r2z archive on Thunderstore and returns its code.
// Anyone with the code can download the profile.
func (c *Client) UploadProfile(ctx context.Context, archive []byte) (string, error) {
	if len(archive) > MaxProfileCodeSize {
		return "", ErrProfileTooLarge
	}
	payload := profileDataPrefix + "\n" + base64.StdEncoding.EncodeToString(archive)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/experimental/legacyprofile/create/", strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload profile: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", errors.New("upload profile: rate limited by Thunderstore, try again later")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("upload profile: %s", resp.Status)
	}
	var body struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || !ValidProfileCode(body.Key) {
		return "", errors.New("upload profile: unexpected response")
	}
	return body.Key, nil
}

// DownloadProfile fetches the .r2z archive behind a profile code.
func (c *Client) DownloadProfile(ctx context.Context, code string) ([]byte, error) {
	code = strings.TrimSpace(code)
	if !ValidProfileCode(code) {
		return nil, ErrInvalidCode
	}
	body, err := c.getURL(ctx, c.baseURL+"/api/experimental/legacyprofile/get/"+code+"/")
	if errors.Is(err, ErrNotFound) {
		return nil, errors.New("profile code not found or expired")
	}
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, 2*MaxProfileCodeSize))
	if err != nil {
		return nil, err
	}
	b64, ok := bytes.CutPrefix(bytes.TrimSpace(data), []byte(profileDataPrefix))
	if !ok {
		return nil, errors.New("profile code does not contain an r2modman profile")
	}
	return base64.StdEncoding.DecodeString(string(bytes.TrimSpace(b64)))
}
