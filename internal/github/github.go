// Package github fetches mod releases from GitHub repositories.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const DefaultAPIURL = "https://api.github.com"

var (
	ErrInvalidRepo = errors.New("enter a GitHub repository, e.g. owner/repo or https://github.com/owner/repo")
	ErrNoReleases  = errors.New("the repository has no releases with downloadable files")
)

type Asset struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	URL  string `json:"browser_download_url"`
}

type Release struct {
	Tag        string    `json:"tag_name"`
	Name       string    `json:"name"`
	Prerelease bool      `json:"prerelease"`
	Draft      bool      `json:"draft"`
	Published  time.Time `json:"published_at"`
	Assets     []Asset   `json:"assets"`
}

type Client struct {
	apiURL    string
	userAgent string
	cacheDir  string
	http      *http.Client
}

func NewClient(apiURL, userAgent, cacheDir string) *Client {
	return &Client{
		apiURL:    strings.TrimRight(apiURL, "/"),
		userAgent: userAgent,
		cacheDir:  cacheDir,
		http:      &http.Client{Timeout: 10 * time.Minute},
	}
}

var (
	repoPart = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	repoURL  = regexp.MustCompile(`^(?:https?://)?(?:www\.)?github\.com/([^/]+)/([^/#?]+)`)
)

// ParseRepo accepts "owner/repo" or a github.com URL.
func ParseRepo(input string) (owner, repo string, err error) {
	input = strings.TrimSpace(input)
	if m := repoURL.FindStringSubmatch(input); m != nil {
		owner, repo = m[1], m[2]
	} else if o, r, ok := strings.Cut(input, "/"); ok && !strings.Contains(r, "/") {
		owner, repo = o, r
	}
	repo = strings.TrimSuffix(repo, ".git")
	if !repoPart.MatchString(owner) || !repoPart.MatchString(repo) || owner == ".." || repo == ".." {
		return "", "", ErrInvalidRepo
	}
	return owner, repo, nil
}

// RepoURL is the canonical web URL of a repository.
func RepoURL(owner, repo string) string {
	return "https://github.com/" + owner + "/" + repo
}

// Releases lists published releases with at least one asset, newest first.
func (c *Client) Releases(ctx context.Context, owner, repo string) ([]Release, error) {
	u := c.apiURL + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/releases?per_page=30"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: %w", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("repository %s/%s not found", owner, repo)
	case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests:
		return nil, errors.New("GitHub rate limit reached, try again later")
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("github: %s", resp.Status)
	}
	var all []Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&all); err != nil {
		return nil, fmt.Errorf("github: %w", err)
	}
	releases := []Release{}
	for _, r := range all {
		if !r.Draft && len(r.Assets) > 0 {
			releases = append(releases, r)
		}
	}
	if len(releases) == 0 {
		return nil, ErrNoReleases
	}
	return releases, nil
}

// LatestRelease returns the newest non-prerelease release with assets.
func (c *Client) LatestRelease(ctx context.Context, owner, repo string) (Release, error) {
	releases, err := c.Releases(ctx, owner, repo)
	if err != nil {
		return Release{}, err
	}
	for _, r := range releases {
		if !r.Prerelease {
			return r, nil
		}
	}
	return Release{}, ErrNoReleases
}

// Download stores a release asset in the cache and returns its path.
func (c *Client) Download(ctx context.Context, owner, repo, tag string, asset Asset, progress func(done, total int64)) (string, error) {
	name := filepath.Base(asset.Name)
	if name != asset.Name || name == "." || name == ".." || !strings.HasPrefix(asset.URL, "https://github.com/") {
		return "", fmt.Errorf("unexpected release asset %q", asset.Name)
	}
	dir := filepath.Join(c.cacheDir, owner, repo, url.PathEscape(tag))
	dest := filepath.Join(dir, name)
	if fi, err := os.Stat(dest); err == nil && fi.Size() == asset.Size {
		return dest, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: %s", name, resp.Status)
	}
	tmp, err := os.CreateTemp(dir, name+".*.part")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	var body io.Reader = io.LimitReader(resp.Body, 2<<30)
	if progress != nil {
		body = &progressReader{r: body, total: resp.ContentLength, fn: progress}
	}
	if _, err := io.Copy(tmp, body); err != nil {
		tmp.Close()
		return "", fmt.Errorf("download %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	return dest, os.Rename(tmp.Name(), dest)
}

type progressReader struct {
	r        io.Reader
	done     int64
	total    int64
	fn       func(done, total int64)
	lastEmit time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.done += int64(n)
	if err == io.EOF || time.Since(p.lastEmit) > 100*time.Millisecond {
		p.lastEmit = time.Now()
		p.fn(p.done, p.total)
	}
	return n, err
}
