// Package library manages games and their profiles on disk.
//
// Layout under the library root:
//
//	cache/<mod-id>/<version>.zip
//	games/<game-id>/game.json
//	games/<game-id>/profiles/<profile-id>/profile.json
//	games/<game-id>/profiles/<profile-id>/BepInEx/...
//
// Directory names are stable ids; display names live in the JSON files, so
// renaming never moves anything on disk.
package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrLastProfile   = errors.New("cannot remove the last profile of a game")
	ErrEmptyName     = errors.New("name must not be empty")
	ErrInvalidPath   = errors.New("game path must be an existing directory")
	ErrAlreadyExists = errors.New("already added")
)

const DefaultProfileName = "Default"

type Library struct {
	root string
	mu   sync.Mutex
}

func New(root string) (*Library, error) {
	for _, dir := range []string{root, filepath.Join(root, "games"), filepath.Join(root, "cache")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create library dir: %w", err)
		}
	}
	return &Library{root: root}, nil
}

func (l *Library) gamesDir() string         { return filepath.Join(l.root, "games") }
func (l *Library) gameDir(id string) string { return filepath.Join(l.gamesDir(), id) }
func (l *Library) profilesDir(gameID string) string {
	return filepath.Join(l.gameDir(gameID), "profiles")
}
func (l *Library) profileDir(gameID, profileID string) string {
	return filepath.Join(l.profilesDir(gameID), profileID)
}

// ListGames returns all games sorted by name.
func (l *Library) ListGames() ([]Game, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	ids, err := listDirs(l.gamesDir())
	if err != nil {
		return nil, err
	}
	games := make([]Game, 0, len(ids))
	for _, id := range ids {
		g, err := l.loadGame(id)
		if errors.Is(err, ErrNotFound) {
			continue // stray directory without game.json
		}
		if err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	sort.Slice(games, func(i, j int) bool { return games[i].Name < games[j].Name })
	return games, nil
}

func (l *Library) GetGame(id string) (Game, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.loadGame(id)
}

// AddGame registers a game installed at path and creates its default profile.
// The game directory itself is only inspected, never modified.
func (l *Library) AddGame(name, path string) (Game, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if name == "" {
		return Game{}, ErrEmptyName
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return Game{}, err
	}
	if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
		return Game{}, ErrInvalidPath
	}

	ids, err := listDirs(l.gamesDir())
	if err != nil {
		return Game{}, err
	}
	for _, id := range ids {
		if g, err := l.loadGame(id); err == nil && g.Path == path {
			return Game{}, fmt.Errorf("game at %s: %w", path, ErrAlreadyExists)
		}
	}

	g := Game{
		ID:      uniqueID(slugify(name, "game"), ids),
		Name:    name,
		Path:    path,
		Runtime: DetectRuntime(path),
	}
	p, err := l.createProfile(g.ID, DefaultProfileName)
	if err != nil {
		_ = os.RemoveAll(l.gameDir(g.ID))
		return Game{}, err
	}
	g.ActiveProfile = p.ID
	if err := l.saveGame(g); err != nil {
		_ = os.RemoveAll(l.gameDir(g.ID))
		return Game{}, err
	}
	return g, nil
}

func (l *Library) RenameGame(id, name string) (Game, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if name == "" {
		return Game{}, ErrEmptyName
	}
	g, err := l.loadGame(id)
	if err != nil {
		return Game{}, err
	}
	g.Name = name
	return g, l.saveGame(g)
}

// RemoveGame deletes the game entry together with all its profiles and installed mods.
func (l *Library) RemoveGame(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.loadGame(id); err != nil {
		return err
	}
	return os.RemoveAll(l.gameDir(id))
}

func (l *Library) SetActiveProfile(gameID, profileID string) (Game, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	g, err := l.loadGame(gameID)
	if err != nil {
		return Game{}, err
	}
	if _, err := l.loadProfile(gameID, profileID); err != nil {
		return Game{}, err
	}
	g.ActiveProfile = profileID
	return g, l.saveGame(g)
}

// ListProfiles returns the profiles of a game sorted by name.
func (l *Library) ListProfiles(gameID string) ([]Profile, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.loadGame(gameID); err != nil {
		return nil, err
	}
	return l.listProfiles(gameID)
}

func (l *Library) GetProfile(gameID, profileID string) (Profile, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.loadProfile(gameID, profileID)
}

func (l *Library) CreateProfile(gameID, name string) (Profile, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if name == "" {
		return Profile{}, ErrEmptyName
	}
	if _, err := l.loadGame(gameID); err != nil {
		return Profile{}, err
	}
	return l.createProfile(gameID, name)
}

func (l *Library) RenameProfile(gameID, profileID, name string) (Profile, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if name == "" {
		return Profile{}, ErrEmptyName
	}
	p, err := l.loadProfile(gameID, profileID)
	if err != nil {
		return Profile{}, err
	}
	p.Name = name
	return p, l.saveProfile(gameID, p)
}

// RemoveProfile deletes a profile with all its mods. The last profile of a game
// cannot be removed; if the active profile is removed, another one becomes active.
func (l *Library) RemoveProfile(gameID, profileID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	g, err := l.loadGame(gameID)
	if err != nil {
		return err
	}
	if _, err := l.loadProfile(gameID, profileID); err != nil {
		return err
	}
	profiles, err := l.listProfiles(gameID)
	if err != nil {
		return err
	}
	if len(profiles) <= 1 {
		return ErrLastProfile
	}
	if err := os.RemoveAll(l.profileDir(gameID, profileID)); err != nil {
		return err
	}
	if g.ActiveProfile == profileID {
		for _, p := range profiles {
			if p.ID != profileID {
				g.ActiveProfile = p.ID
				break
			}
		}
		return l.saveGame(g)
	}
	return nil
}

func (l *Library) createProfile(gameID, name string) (Profile, error) {
	ids, err := listDirs(l.profilesDir(gameID))
	if err != nil {
		return Profile{}, err
	}
	p := Profile{
		SchemaVersion: ProfileSchemaVersion,
		ID:            uniqueID(slugify(name, "profile"), ids),
		Name:          name,
		Mods:          []Mod{},
	}
	dir := l.profileDir(gameID, p.ID)
	for _, sub := range []string{"plugins", "patchers", "config"} {
		if err := os.MkdirAll(filepath.Join(dir, "BepInEx", sub), 0o755); err != nil {
			return Profile{}, err
		}
	}
	if err := l.saveProfile(gameID, p); err != nil {
		_ = os.RemoveAll(dir)
		return Profile{}, err
	}
	return p, nil
}

func (l *Library) listProfiles(gameID string) ([]Profile, error) {
	ids, err := listDirs(l.profilesDir(gameID))
	if err != nil {
		return nil, err
	}
	profiles := make([]Profile, 0, len(ids))
	for _, id := range ids {
		p, err := l.loadProfile(gameID, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })
	return profiles, nil
}

func (l *Library) loadGame(id string) (Game, error) {
	if !validID(id) {
		return Game{}, fmt.Errorf("game %q: %w", id, ErrNotFound)
	}
	var g Game
	if err := readJSON(filepath.Join(l.gameDir(id), "game.json"), &g); err != nil {
		return Game{}, fmt.Errorf("game %q: %w", id, err)
	}
	g.ID = id
	return g, nil
}

func (l *Library) saveGame(g Game) error {
	return writeJSON(filepath.Join(l.gameDir(g.ID), "game.json"), g)
}

func (l *Library) loadProfile(gameID, id string) (Profile, error) {
	if !validID(gameID) || !validID(id) {
		return Profile{}, fmt.Errorf("profile %q: %w", id, ErrNotFound)
	}
	var p Profile
	if err := readJSON(filepath.Join(l.profileDir(gameID, id), "profile.json"), &p); err != nil {
		return Profile{}, fmt.Errorf("profile %q: %w", id, err)
	}
	p.ID = id
	if p.Mods == nil {
		p.Mods = []Mod{}
	}
	return p, nil
}

func (l *Library) saveProfile(gameID string, p Profile) error {
	return writeJSON(filepath.Join(l.profileDir(gameID, p.ID), "profile.json"), p)
}
