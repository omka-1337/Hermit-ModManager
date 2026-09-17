package profileshare

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"bepinexmodmanager/internal/library"
	"bepinexmodmanager/internal/modinstall"
	"bepinexmodmanager/internal/thunderstore"
)

// CodeStore stores profiles behind shareable codes (Thunderstore).
type CodeStore interface {
	UploadProfile(ctx context.Context, archive []byte) (string, error)
	DownloadProfile(ctx context.Context, code string) ([]byte, error)
}

type Sharer struct {
	lib       *library.Library
	installer *modinstall.Installer
	codes     CodeStore
	// importDir keeps archives downloaded from codes between preview and import.
	importDir string
}

func NewSharer(lib *library.Library, installer *modinstall.Installer, codes CodeStore, importDir string) *Sharer {
	return &Sharer{lib: lib, installer: installer, codes: codes, importDir: importDir}
}

// Source is a profile to import: an .r2z file or a profile code.
type Source struct {
	File string `json:"file"`
	Code string `json:"code"`
}

type Preview struct {
	// Archive is the local .r2z file to pass to Import.
	Archive     string      `json:"archive"`
	ProfileName string      `json:"profileName"`
	Mods        []ExportMod `json:"mods"`
}

type ImportProgress struct {
	GameID    string `json:"gameId"`
	ProfileID string `json:"profileId"`
	// Package is the mod being installed, "<author>-<name>-<version>".
	Package string `json:"package"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
}

type FailedMod struct {
	Package string `json:"package"`
	Error   string `json:"error"`
}

type ImportResult struct {
	Profile library.Profile `json:"profile"`
	// Failed lists mods that could not be installed, e.g. removed from Thunderstore.
	Failed []FailedMod `json:"failed"`
}

// Preview reads a profile to import without changing anything.
func (s *Sharer) Preview(ctx context.Context, src Source) (Preview, error) {
	archive := src.File
	if src.Code != "" {
		data, err := s.codes.DownloadProfile(ctx, src.Code)
		if err != nil {
			return Preview{}, err
		}
		if err := os.MkdirAll(s.importDir, 0o755); err != nil {
			return Preview{}, err
		}
		archive = filepath.Join(s.importDir, "code-import.r2z")
		if err := os.WriteFile(archive, data, 0o644); err != nil {
			return Preview{}, err
		}
	}
	if archive == "" {
		return Preview{}, errors.New("choose a file or enter a profile code")
	}
	export, err := ReadExport(archive)
	if err != nil {
		return Preview{}, err
	}
	return Preview{Archive: archive, ProfileName: export.ProfileName, Mods: export.Mods}, nil
}

// Import creates a new profile from an .r2z archive the way r2modman does:
// the listed mods are installed in their exact versions, disabled ones are
// disabled, then the config files are copied over. Mods that fail to install
// are reported and skipped.
func (s *Sharer) Import(ctx context.Context, gameID, archive, name string, onProgress func(ImportProgress), onInstall func(modinstall.Progress)) (ImportResult, error) {
	export, err := ReadExport(archive)
	if err != nil {
		return ImportResult{}, err
	}
	profile, err := s.lib.CreateProfile(gameID, name)
	if err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{Failed: []FailedMod{}}
	fail := func(pkg string, err error) {
		result.Failed = append(result.Failed, FailedMod{Package: pkg, Error: err.Error()})
	}

	// The exported list already contains every dependency in the version the
	// profile had, and those versions win over the ones in package manifests.
	pinned := map[string]string{}
	for _, m := range export.Mods {
		pinned[m.Name] = m.Version.String()
	}

	for i, m := range export.Mods {
		pkg := m.Name + "-" + m.Version.String()
		if onProgress != nil {
			onProgress(ImportProgress{GameID: gameID, ProfileID: profile.ID, Package: pkg, Done: i, Total: len(export.Mods)})
		}
		if err := ctx.Err(); err != nil {
			return ImportResult{}, err
		}
		ref, err := m.Ref()
		if err != nil {
			fail(pkg, err)
			continue
		}
		current, err := s.lib.GetProfile(gameID, profile.ID)
		if err != nil {
			return ImportResult{}, err
		}
		if installedVersion(current, ref.ID()) == ref.Version {
			continue // already installed as a dependency of an earlier mod
		}
		opts := modinstall.Options{Modpack: true, Pinned: pinned}
		if _, err := s.installer.Install(ctx, gameID, profile.ID, ref, opts, onInstall); err != nil {
			fail(pkg, err)
		}
	}

	for _, m := range export.Mods {
		if m.Enabled {
			continue
		}
		if _, err := s.installer.SetEnabled(gameID, profile.ID, m.Name, false); err != nil && !errors.Is(err, library.ErrNotFound) {
			fail(m.Name+"-"+m.Version.String(), fmt.Errorf("disable: %w", err))
		}
	}

	profileDir, err := s.lib.ProfileDir(gameID, profile.ID)
	if err != nil {
		return ImportResult{}, err
	}
	if err := ExtractConfigs(archive, profileDir); err != nil {
		fail("config files", err)
	}
	result.Profile, err = s.installer.Refresh(gameID, profile.ID)
	if onProgress != nil {
		onProgress(ImportProgress{GameID: gameID, ProfileID: profile.ID, Done: len(export.Mods), Total: len(export.Mods)})
	}
	return result, err
}

func installedVersion(p library.Profile, id string) string {
	for _, m := range p.Mods {
		if m.ID == id {
			return m.Version
		}
	}
	return ""
}

func (s *Sharer) archive(gameID, profileID string) ([]byte, error) {
	profile, err := s.lib.GetProfile(gameID, profileID)
	if err != nil {
		return nil, err
	}
	profileDir, err := s.lib.ProfileDir(gameID, profileID)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := WriteR2Z(&buf, profile, profileDir); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExportFile writes a profile as an .r2z file.
func (s *Sharer) ExportFile(gameID, profileID, dest string) error {
	data, err := s.archive(gameID, profileID)
	if err != nil {
		return err
	}
	tmp := dest + ".part"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

// ExportCode uploads a profile to Thunderstore and returns its code.
func (s *Sharer) ExportCode(ctx context.Context, gameID, profileID string) (string, error) {
	data, err := s.archive(gameID, profileID)
	if err != nil {
		return "", err
	}
	if len(data) > thunderstore.MaxProfileCodeSize {
		return "", fmt.Errorf("%w (%d MB, limit %d MB); export it as a file instead",
			thunderstore.ErrProfileTooLarge, len(data)>>20, thunderstore.MaxProfileCodeSize/1_000_000)
	}
	return s.codes.UploadProfile(ctx, data)
}
