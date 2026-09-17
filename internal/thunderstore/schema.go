package thunderstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const schemaTTL = 24 * time.Hour

// Ecosystem is the subset of Thunderstore's ecosystem schema we use: it maps
// games (by Steam app id or executable) to Thunderstore communities.
type Ecosystem struct {
	Games             map[string]EcosystemGame `json:"games"`
	ModloaderPackages []ModloaderPackage       `json:"modloaderPackages"`
}

// ModloaderPackage identifies a mod loader package (e.g. BepInExPack) and the
// folder inside its archive that maps to the profile root.
type ModloaderPackage struct {
	PackageID  string `json:"packageId"`
	RootFolder string `json:"rootFolder"`
	Loader     string `json:"loader"`
}

type EcosystemGame struct {
	Label string `json:"label"`
	Meta  struct {
		DisplayName string `json:"displayName"`
	} `json:"meta"`
	Distributions []Distribution `json:"distributions"`
	R2modman      []GameSettings `json:"r2modman"`
	Thunderstore  *struct {
		DisplayName string `json:"displayName"`
	} `json:"thunderstore"`
}

type Distribution struct {
	Platform   string `json:"platform"`
	Identifier string `json:"identifier"`
}

type GameSettings struct {
	// PackageIndex is the listing URL of the game's community, e.g.
	// https://thunderstore.io/c/valheim/api/v1/package-listing-index/.
	PackageIndex           string         `json:"packageIndex"`
	ExeNames               []string       `json:"exeNames"`
	DataFolderName         string         `json:"dataFolderName"`
	PackageLoader          string         `json:"packageLoader"`
	Distributions          []Distribution `json:"distributions"`
	InstallRules           []InstallRule  `json:"installRules"`
	RelativeFileExclusions []string       `json:"relativeFileExclusions"`
}

type TrackingMethod string

const (
	// TrackingSubdir installs into <route>/<author>-<name>/.
	TrackingSubdir TrackingMethod = "subdir"
	// TrackingState installs directly into <route>, tracking each file.
	TrackingState TrackingMethod = "state"
	// TrackingNone installs directly into <route> without tracking (configs).
	TrackingNone TrackingMethod = "none"
)

// InstallRule says where package files go in a profile, as used by r2modman.
type InstallRule struct {
	Route                 string         `json:"route"`
	TrackingMethod        TrackingMethod `json:"trackingMethod"`
	DefaultFileExtensions []string       `json:"defaultFileExtensions"`
	IsDefaultLocation     bool           `json:"isDefaultLocation"`
	SubRoutes             []InstallRule  `json:"subRoutes"`
}

// Community is a Thunderstore community (one per game).
type Community struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Schema returns the ecosystem schema, cached in memory and on disk for a day.
// If refreshing fails, a stale cached copy is used.
func (c *Client) Schema(ctx context.Context) (*Ecosystem, error) {
	c.schemaMu.Lock()
	defer c.schemaMu.Unlock()

	if c.schema != nil && time.Since(c.schemaFetched) < schemaTTL {
		return c.schema, nil
	}
	data, err := c.cachedFetch(ctx, "ecosystem-schema.json", c.baseURL+"/api/experimental/schema/dev/latest/", func(b []byte) error {
		return json.Unmarshal(b, &Ecosystem{})
	})
	if err != nil {
		return nil, err
	}
	var schema Ecosystem
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}
	c.schema, c.schemaFetched = &schema, time.Now()
	return &schema, nil
}

// cachedFetch returns a document from the disk cache if younger than
// schemaTTL, otherwise downloads it. validate rejects broken downloads. When
// the download fails, a stale cached copy is returned if there is one.
func (c *Client) cachedFetch(ctx context.Context, cacheName, fullURL string, validate func([]byte) error) ([]byte, error) {
	cachePath := filepath.Join(c.cacheDir, cacheName)
	cached, cacheErr := os.ReadFile(cachePath)
	if cacheErr == nil && validate(cached) != nil {
		cached, cacheErr = nil, errors.New("invalid cache")
	}
	if fi, err := os.Stat(cachePath); cacheErr == nil && err == nil && time.Since(fi.ModTime()) < schemaTTL {
		return cached, nil
	}

	data, err := c.fetchURL(ctx, fullURL)
	if err == nil {
		err = validate(data)
	}
	if err != nil {
		if cacheErr == nil {
			return cached, nil
		}
		return nil, err
	}
	if err := os.MkdirAll(c.cacheDir, 0o755); err == nil {
		_ = os.WriteFile(cachePath, data, 0o644)
	}
	return data, nil
}

// FindCommunity returns the BepInEx community for a game, matched by Steam app
// id first and then by executable name.
func (e *Ecosystem) FindCommunity(steamAppID, executable string) (Community, bool) {
	c, _, ok := e.FindGame(steamAppID, executable)
	return c, ok
}

// FindGame is FindCommunity that also returns the game's manager settings,
// including install rules.
func (e *Ecosystem) FindGame(steamAppID, executable string) (Community, GameSettings, bool) {
	type match struct {
		community Community
		settings  GameSettings
	}
	var byExe *match
	for label, g := range e.Games {
		for _, s := range g.R2modman {
			if s.PackageLoader != "bepinex" {
				continue
			}
			m := match{communityOf(label, g, s), s}
			if steamAppID != "" && (hasSteamID(s.Distributions, steamAppID) || hasSteamID(g.Distributions, steamAppID)) {
				return m.community, m.settings, true
			}
			if byExe == nil && executable != "" {
				for _, exe := range s.ExeNames {
					if strings.EqualFold(exe, executable) {
						byExe = &m
					}
				}
			}
		}
	}
	if byExe != nil {
		return byExe.community, byExe.settings, true
	}
	return Community{}, GameSettings{}, false
}

// communityOf names the Thunderstore community of a game. Older games in the
// schema have no "thunderstore" section, so like r2modman the community id is
// taken from the package index URL.
func communityOf(label string, g EcosystemGame, s GameSettings) Community {
	c := Community{ID: label, Name: g.Meta.DisplayName}
	if g.Thunderstore != nil && g.Thunderstore.DisplayName != "" {
		c.Name = g.Thunderstore.DisplayName
	}
	if m := packageIndexCommunity.FindStringSubmatch(s.PackageIndex); m != nil {
		c.ID = m[1]
	}
	if c.Name == "" {
		c.Name = label
	}
	return c
}

var packageIndexCommunity = regexp.MustCompile(`/c/([^/]+)/`)

func hasSteamID(ds []Distribution, id string) bool {
	for _, d := range ds {
		if d.Platform == "steam" && d.Identifier == id {
			return true
		}
	}
	return false
}
