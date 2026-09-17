package thunderstore

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

type Ordering string

const (
	OrderLastUpdated    Ordering = "last-updated"
	OrderNewest         Ordering = "newest"
	OrderMostDownloaded Ordering = "most-downloaded"
	OrderTopRated       Ordering = "top-rated"
)

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Section struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Priority int    `json:"priority"`
}

type Filters struct {
	Categories []Category `json:"categories"`
	Sections   []Section  `json:"sections"`
}

type ListOptions struct {
	Query    string   `json:"query"`
	Ordering Ordering `json:"ordering"`
	// Section is a section UUID from Filters; empty means all packages.
	Section string `json:"section"`
	Page    int    `json:"page"`
	// IncludeNSFW is decided by the manager settings, not by the caller.
	IncludeNSFW bool `json:"-"`
}

type PackageSummary struct {
	Namespace     string     `json:"namespace"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	IconURL       string     `json:"icon_url"`
	DownloadCount int64      `json:"download_count"`
	RatingCount   int        `json:"rating_count"`
	Size          int64      `json:"size"`
	LastUpdated   time.Time  `json:"last_updated"`
	IsPinned      bool       `json:"is_pinned"`
	IsDeprecated  bool       `json:"is_deprecated"`
	IsNSFW        bool       `json:"is_nsfw"`
	Categories    []Category `json:"categories"`
}

type PackageList struct {
	Count    int              `json:"count"`
	Page     int              `json:"page"`
	HasMore  bool             `json:"has_more"`
	Packages []PackageSummary `json:"packages"`
}

type Dependency struct {
	Namespace     string `json:"namespace"`
	Name          string `json:"name"`
	VersionNumber string `json:"version_number"`
	Description   string `json:"description"`
	IconURL       string `json:"icon_url"`
	IsUnavailable bool   `json:"is_unavailable"`
}

type PackageDetail struct {
	Namespace      string       `json:"namespace"`
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	IconURL        string       `json:"icon_url"`
	WebsiteURL     string       `json:"website_url"`
	DownloadCount  int64        `json:"download_count"`
	RatingCount    int          `json:"rating_count"`
	Size           int64        `json:"size"`
	LatestVersion  string       `json:"latest_version_number"`
	VersionCount   int          `json:"version_count"`
	VersionCreated time.Time    `json:"version_created"`
	DownloadURL    string       `json:"download_url"`
	IsDeprecated   bool         `json:"is_deprecated"`
	IsNSFW         bool         `json:"is_nsfw"`
	Categories     []Category   `json:"categories"`
	Dependencies   []Dependency `json:"dependencies"`
	DependantCount int          `json:"dependant_count"`
	// PageURL is the package page on the Thunderstore website.
	PageURL string `json:"page_url"`
}

type Version struct {
	VersionNumber string    `json:"version_number"`
	Created       time.Time `json:"datetime_created"`
	DownloadCount int64     `json:"download_count"`
	DownloadURL   string    `json:"download_url"`
}

func (c *Client) Filters(ctx context.Context, community string) (Filters, error) {
	var resp struct {
		Categories []Category `json:"package_categories"`
		Sections   []Section  `json:"sections"`
	}
	path := "/api/cyberstorm/community/" + url.PathEscape(community) + "/filters/"
	if err := c.getJSON(ctx, path, nil, &resp); err != nil {
		return Filters{}, err
	}
	return Filters{Categories: resp.Categories, Sections: resp.Sections}, nil
}

// ListPackages returns one page (20 packages) of a community listing.
// Deprecated packages are excluded; NSFW ones only with IncludeNSFW.
func (c *Client) ListPackages(ctx context.Context, community string, opts ListOptions) (PackageList, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Ordering == "" {
		opts.Ordering = OrderMostDownloaded
	}
	q := url.Values{
		"ordering":   {string(opts.Ordering)},
		"page":       {strconv.Itoa(opts.Page)},
		"deprecated": {"False"},
		"nsfw":       {pyBool(opts.IncludeNSFW)},
	}
	if opts.Query != "" {
		q.Set("q", opts.Query)
	}
	if opts.Section != "" {
		q.Set("section", opts.Section)
	}
	var resp struct {
		Count   int              `json:"count"`
		Next    *string          `json:"next"`
		Results []PackageSummary `json:"results"`
	}
	path := "/api/cyberstorm/listing/" + url.PathEscape(community) + "/"
	if err := c.getJSON(ctx, path, q, &resp); err != nil {
		return PackageList{}, err
	}
	if resp.Results == nil {
		resp.Results = []PackageSummary{}
	}
	return PackageList{Count: resp.Count, Page: opts.Page, HasMore: resp.Next != nil, Packages: resp.Results}, nil
}

func (c *Client) Package(ctx context.Context, community, namespace, name string) (PackageDetail, error) {
	var d PackageDetail
	path := "/api/cyberstorm/listing/" + url.PathEscape(community) + "/" + url.PathEscape(namespace) + "/" + url.PathEscape(name) + "/"
	if err := c.getJSON(ctx, path, nil, &d); err != nil {
		return PackageDetail{}, err
	}
	d.PageURL = c.baseURL + "/c/" + url.PathEscape(community) + "/p/" + url.PathEscape(namespace) + "/" + url.PathEscape(name) + "/"
	return d, nil
}

// Versions returns all versions of a package, in the order returned by the API.
func (c *Client) Versions(ctx context.Context, namespace, name string) ([]Version, error) {
	var versions []Version
	path := "/api/cyberstorm/package/" + url.PathEscape(namespace) + "/" + url.PathEscape(name) + "/versions/"
	if err := c.getJSON(ctx, path, nil, &versions); err != nil {
		return nil, err
	}
	return versions, nil
}

// Readme returns the README markdown of a package version.
func (c *Client) Readme(ctx context.Context, namespace, name, version string) (string, error) {
	var resp struct {
		Markdown string `json:"markdown"`
	}
	path := "/api/experimental/package/" + url.PathEscape(namespace) + "/" + url.PathEscape(name) + "/" + url.PathEscape(version) + "/readme/"
	if err := c.getJSON(ctx, path, nil, &resp); err != nil {
		return "", err
	}
	return resp.Markdown, nil
}

// pyBool formats a bool the way the Django-based API expects.
func pyBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}
