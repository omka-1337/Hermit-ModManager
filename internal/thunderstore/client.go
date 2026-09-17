// Package thunderstore is a client for the Thunderstore mod repository.
package thunderstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const DefaultBaseURL = "https://thunderstore.io"

var ErrNotFound = errors.New("not found on Thunderstore")

type Client struct {
	baseURL   string
	userAgent string
	http      *http.Client
	download  *http.Client
	cacheDir  string

	schemaMu      sync.Mutex
	schema        *Ecosystem
	schemaFetched time.Time
}

// NewClient creates a client; the ecosystem schema is cached in cacheDir.
func NewClient(baseURL, userAgent, cacheDir string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		userAgent: userAgent,
		http:      &http.Client{Timeout: 30 * time.Second},
		// Package archives can be hundreds of MB, so only the time to first
		// byte is limited; cancellation goes through the request context.
		download: &http.Client{Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ResponseHeaderTimeout: 30 * time.Second,
		}},
		cacheDir: cacheDir,
	}
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, v any) error {
	body, err := c.get(ctx, path, query)
	if err != nil {
		return err
	}
	defer body.Close()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, query url.Values) (io.ReadCloser, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("thunderstore request failed: %w", err)
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
		resp.Body.Close()
		return nil, ErrNotFound
	case resp.StatusCode != http.StatusOK:
		resp.Body.Close()
		return nil, fmt.Errorf("thunderstore: %s returned %s", path, resp.Status)
	}
	return resp.Body, nil
}
