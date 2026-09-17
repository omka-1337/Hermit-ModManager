package thunderstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const schemaTTL = 24 * time.Hour

// Ecosystem is the subset of Thunderstore's ecosystem schema we use: it maps
// games (by Steam app id or executable) to Thunderstore communities.
type Ecosystem struct {
	Games map[string]EcosystemGame `json:"games"`
}

type EcosystemGame struct {
	Label         string         `json:"label"`
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
	ExeNames       []string       `json:"exeNames"`
	DataFolderName string         `json:"dataFolderName"`
	PackageLoader  string         `json:"packageLoader"`
	Distributions  []Distribution `json:"distributions"`
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
	var byExe *Community
	for label, g := range e.Games {
		if g.Thunderstore == nil {
			continue
		}
		for _, s := range g.R2modman {
			if s.PackageLoader != "bepinex" {
				continue
			}
			community := Community{ID: label, Name: g.Thunderstore.DisplayName}
			if steamAppID != "" && (hasSteamID(s.Distributions, steamAppID) || hasSteamID(g.Distributions, steamAppID)) {
				return community, true
			}
			if byExe == nil && executable != "" {
				for _, exe := range s.ExeNames {
					if strings.EqualFold(exe, executable) {
						byExe = &community
					}
				}
			}
		}
	}
	if byExe != nil {
		return *byExe, true
	}
	return Community{}, false
}

func hasSteamID(ds []Distribution, id string) bool {
	for _, d := range ds {
		if d.Platform == "steam" && d.Identifier == id {
			return true
		}
	}
	return false
}
