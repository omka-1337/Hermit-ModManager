package modinstall

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"bepinexmodmanager/internal/library"
	"bepinexmodmanager/internal/thunderstore"
)

type Stage string

const (
	StageDownload Stage = "download"
	StageInstall  Stage = "install"
)

type Progress struct {
	GameID    string `json:"gameId"`
	ProfileID string `json:"profileId"`
	// Target is the package the user asked to install ("<author>-<name>").
	Target string `json:"target"`
	// Package is the package currently being processed, a dependency or the target.
	Package string `json:"package"`
	Stage   Stage  `json:"stage"`
	Done    int64  `json:"done"`
	Total   int64  `json:"total"`
}

type Downloader interface {
	DownloadPackage(ctx context.Context, ref thunderstore.PackageRef, progress func(done, total int64)) (thunderstore.Archive, error)
}

type Installer struct {
	lib        *library.Library
	downloader Downloader

	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewInstaller(lib *library.Library, downloader Downloader) *Installer {
	return &Installer{lib: lib, downloader: downloader, locks: map[string]*sync.Mutex{}}
}

// profileLock serialises installs and uninstalls within one profile.
func (in *Installer) profileLock(gameID, profileID string) *sync.Mutex {
	in.mu.Lock()
	defer in.mu.Unlock()
	key := gameID + "/" + profileID
	if in.locks[key] == nil {
		in.locks[key] = &sync.Mutex{}
	}
	return in.locks[key]
}

type plannedPackage struct {
	ref      thunderstore.PackageRef
	archive  thunderstore.Archive
	manifest thunderstore.Manifest
}

// Install installs a package version together with its dependencies.
// Dependency versions are minimums: an installed dependency is kept if it is
// the same or newer, otherwise the required version is installed.
func (in *Installer) Install(ctx context.Context, gameID, profileID string, ref thunderstore.PackageRef, onProgress func(Progress)) (library.Profile, error) {
	if err := ref.Validate(); err != nil {
		return library.Profile{}, err
	}
	lock := in.profileLock(gameID, profileID)
	lock.Lock()
	defer lock.Unlock()

	profile, err := in.lib.GetProfile(gameID, profileID)
	if err != nil {
		return library.Profile{}, err
	}
	profileDir, err := in.lib.ProfileDir(gameID, profileID)
	if err != nil {
		return library.Profile{}, err
	}
	installed := map[string]string{}
	for _, m := range profile.Mods {
		installed[m.ID] = m.Version
	}

	report := func(pkg thunderstore.PackageRef, stage Stage, done, total int64) {
		if onProgress != nil {
			onProgress(Progress{GameID: gameID, ProfileID: profileID, Target: ref.ID(), Package: pkg.String(), Stage: stage, Done: done, Total: total})
		}
	}

	// Resolve the dependency tree depth-first so dependencies come before dependants.
	var plan []plannedPackage
	planned := map[string]int{}
	var visit func(r thunderstore.PackageRef, root bool) error
	visit = func(r thunderstore.PackageRef, root bool) error {
		if i, ok := planned[r.ID()]; ok {
			if i < 0 || CompareVersions(plan[i].ref.Version, r.Version) >= 0 {
				return nil // being resolved (cycle) or already planned at a sufficient version
			}
		}
		if v, ok := installed[r.ID()]; ok && !root && CompareVersions(v, r.Version) >= 0 {
			return nil
		}
		planned[r.ID()] = -1
		archive, err := in.downloader.DownloadPackage(ctx, r, func(done, total int64) {
			report(r, StageDownload, done, total)
		})
		if err != nil {
			return err
		}
		manifest, err := thunderstore.ReadManifest(archive.Path)
		if err != nil {
			return fmt.Errorf("%s: %w", r, err)
		}
		for _, dep := range manifest.Dependencies {
			depRef, err := thunderstore.ParseDependency(dep)
			if err != nil {
				return fmt.Errorf("%s: %w", r, err)
			}
			if err := visit(depRef, false); err != nil {
				return err
			}
		}
		plan = append(plan, plannedPackage{ref: r, archive: archive, manifest: manifest})
		planned[r.ID()] = len(plan) - 1
		return nil
	}
	if err := visit(ref, true); err != nil {
		return library.Profile{}, err
	}

	for i, p := range plan {
		if planned[p.ref.ID()] != i {
			continue // superseded by a newer version planned later
		}
		if err := ctx.Err(); err != nil {
			return library.Profile{}, err
		}
		if installed[p.ref.ID()] == p.ref.Version && i != len(plan)-1 {
			continue
		}
		report(p.ref, StageInstall, 0, 0)
		profile, err = in.installOne(gameID, profileID, profileDir, p)
		if err != nil {
			return library.Profile{}, err
		}
	}
	return in.lib.GetProfile(gameID, profileID)
}

func (in *Installer) installOne(gameID, profileID, profileDir string, p plannedPackage) (library.Profile, error) {
	current, err := in.lib.GetProfile(gameID, profileID)
	if err != nil {
		return library.Profile{}, err
	}
	if i := slices.IndexFunc(current.Mods, func(m library.Mod) bool { return m.ID == p.ref.ID() }); i >= 0 {
		if err := Remove(profileDir, current.Mods[i].Files); err != nil {
			return library.Profile{}, fmt.Errorf("remove old %s: %w", current.Mods[i].ID, err)
		}
	}

	files, extractErr := Extract(p.archive.Path, profileDir, p.ref.ID())
	return in.lib.UpdateProfile(gameID, profileID, func(prof *library.Profile) error {
		prof.Mods = slices.DeleteFunc(prof.Mods, func(m library.Mod) bool { return m.ID == p.ref.ID() })
		if extractErr != nil {
			return fmt.Errorf("install %s: %w", p.ref, extractErr)
		}
		deps := p.manifest.Dependencies
		if deps == nil {
			deps = []string{}
		}
		prof.Mods = append(prof.Mods, library.Mod{
			ID:      p.ref.ID(),
			Name:    p.ref.Name,
			Author:  p.ref.Namespace,
			Version: p.ref.Version,
			Enabled: true,
			Source: library.ModSource{
				Type:   library.SourceThunderstore,
				URL:    p.archive.URL,
				SHA256: p.archive.SHA256,
			},
			Dependencies: deps,
			Files:        files,
		})
		slices.SortFunc(prof.Mods, func(a, b library.Mod) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
		return nil
	})
}

// Uninstall removes a mod's files and its entry from the profile. Dependencies
// are left installed.
func (in *Installer) Uninstall(gameID, profileID, modID string) (library.Profile, error) {
	lock := in.profileLock(gameID, profileID)
	lock.Lock()
	defer lock.Unlock()

	profileDir, err := in.lib.ProfileDir(gameID, profileID)
	if err != nil {
		return library.Profile{}, err
	}
	var files []string
	profile, err := in.lib.UpdateProfile(gameID, profileID, func(p *library.Profile) error {
		i := slices.IndexFunc(p.Mods, func(m library.Mod) bool { return m.ID == modID })
		if i < 0 {
			return fmt.Errorf("mod %q: %w", modID, library.ErrNotFound)
		}
		files = p.Mods[i].Files
		p.Mods = slices.Delete(p.Mods, i, i+1)
		return nil
	})
	if err != nil {
		return library.Profile{}, err
	}
	return profile, Remove(profileDir, files)
}

// CompareVersions compares dotted numeric versions like "5.4.2100".
func CompareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < max(len(as), len(bs)); i++ {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			return x - y
		}
	}
	return 0
}
