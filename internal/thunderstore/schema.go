package thunderstore

import (
	"context"
	"encoding/json"
	"io"
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
	cachePath := filepath.Join(c.cacheDir, "ecosystem-schema.json")
	cached, cachedAt := readCachedSchema(cachePath)
	if cached != nil && time.Since(cachedAt) < schemaTTL {
		c.schema, c.schemaFetched = cached, cachedAt
		return cached, nil
	}

	body, err := c.get(ctx, "/api/experimental/schema/dev/latest/", nil)
	if err != nil {
		if cached != nil {
			c.schema, c.schemaFetched = cached, time.Now()
			return cached, nil
		}
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var schema Ecosystem
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.cacheDir, 0o755); err == nil {
		_ = os.WriteFile(cachePath, data, 0o644)
	}
	c.schema, c.schemaFetched = &schema, time.Now()
	return &schema, nil
}

func readCachedSchema(path string) (*Ecosystem, time.Time) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}
	}
	var schema Ecosystem
	if json.Unmarshal(data, &schema) != nil {
		return nil, time.Time{}
	}
	return &schema, fi.ModTime()
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
