// Package settings stores manager-wide preferences in config.json.
package settings

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type Settings struct {
	// SetupCompleted is set once the first-run setup is finished or skipped.
	SetupCompleted bool `json:"setupCompleted"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) *Store {
	return &Store{path: filepath.Join(root, "config.json")}
}

func (s *Store) Get() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Store) CompleteSetup() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.load()
	if err != nil {
		return Settings{}, err
	}
	st.SetupCompleted = true
	return st, s.save(st)
}

func (s *Store) load() (Settings, error) {
	var st Settings
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	return st, json.Unmarshal(data, &st)
}

func (s *Store) save(st Settings) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
