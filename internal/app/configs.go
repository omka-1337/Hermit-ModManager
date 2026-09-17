package app

import (
	"strings"

	"bepinexmodmanager/internal/configs"
	"bepinexmodmanager/internal/library"
)

// ConfigService lets the frontend browse and edit mod configs of a profile.
type ConfigService struct {
	lib *library.Library
}

func NewConfigService(lib *library.Library) *ConfigService {
	return &ConfigService{lib: lib}
}

type ConfigContent struct {
	Path string `json:"path"`
	Text string `json:"text"`
	// Document is the parsed structure of .cfg files, nil for other files.
	Document *configs.Document `json:"document"`
}

func (s *ConfigService) ListConfigs(gameID, profileID string) ([]configs.FileInfo, error) {
	dir, err := s.lib.ProfileDir(gameID, profileID)
	if err != nil {
		return nil, err
	}
	return configs.List(dir)
}

func (s *ConfigService) ReadConfig(gameID, profileID, path string) (ConfigContent, error) {
	dir, err := s.lib.ProfileDir(gameID, profileID)
	if err != nil {
		return ConfigContent{}, err
	}
	text, err := configs.ReadText(dir, path)
	if err != nil {
		return ConfigContent{}, err
	}
	return content(path, text), nil
}

// SaveConfigChanges updates individual settings of a .cfg file, leaving the
// rest of the file untouched.
func (s *ConfigService) SaveConfigChanges(gameID, profileID, path string, changes []configs.Change) (ConfigContent, error) {
	dir, err := s.lib.ProfileDir(gameID, profileID)
	if err != nil {
		return ConfigContent{}, err
	}
	text, err := configs.ReadText(dir, path)
	if err != nil {
		return ConfigContent{}, err
	}
	if text, err = configs.ApplyChanges(text, changes); err != nil {
		return ConfigContent{}, err
	}
	if err := configs.WriteText(dir, path, text); err != nil {
		return ConfigContent{}, err
	}
	return content(path, text), nil
}

// SaveConfigText replaces a config file's contents.
func (s *ConfigService) SaveConfigText(gameID, profileID, path, text string) (ConfigContent, error) {
	dir, err := s.lib.ProfileDir(gameID, profileID)
	if err != nil {
		return ConfigContent{}, err
	}
	if err := configs.WriteText(dir, path, text); err != nil {
		return ConfigContent{}, err
	}
	return content(path, text), nil
}

func content(path, text string) ConfigContent {
	c := ConfigContent{Path: path, Text: text}
	if strings.HasSuffix(strings.ToLower(path), ".cfg") {
		doc := configs.ParseCfg(text)
		c.Document = &doc
	}
	return c
}
