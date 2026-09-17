package modinstall

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"hermit/internal/github"
	"hermit/internal/library"
)

// GitHubSource provides mod releases from GitHub.
type GitHubSource interface {
	LatestRelease(ctx context.Context, owner, repo string) (github.Release, error)
	Download(ctx context.Context, owner, repo, tag string, asset github.Asset, progress func(done, total int64)) (string, error)
}

// SetGitHub enables installing and updating mods from GitHub releases.
func (in *Installer) SetGitHub(gh GitHubSource) { in.github = gh }

// InstallGitHub installs a downloaded GitHub release asset (pkg.Path) as a mod.
func (in *Installer) InstallGitHub(ctx context.Context, gameID, profileID string, pkg LocalPackage, owner, repo, tag, asset string, replaceConflicts bool, onProgress func(Progress)) (library.Profile, error) {
	source := library.ModSource{
		Type:    library.SourceGitHub,
		URL:     github.RepoURL(owner, repo),
		Release: tag,
		Asset:   asset,
	}
	return in.installArchive(ctx, gameID, profileID, pkg, source, replaceConflicts, onProgress)
}

func (in *Installer) latestGitHubRelease(ctx context.Context, m library.Mod) (github.Release, error) {
	owner, repo, err := github.ParseRepo(m.Source.URL)
	if err != nil {
		return github.Release{}, err
	}
	return in.github.LatestRelease(ctx, owner, repo)
}

// updateGitHub installs the latest release of a GitHub mod, taking the asset
// with the same name as before, or the only mod file of the release.
func (in *Installer) updateGitHub(ctx context.Context, gameID, profileID string, m library.Mod, onProgress func(Progress)) (library.Profile, error) {
	if in.github == nil {
		return library.Profile{}, fmt.Errorf("GitHub is not available")
	}
	owner, repo, err := github.ParseRepo(m.Source.URL)
	if err != nil {
		return library.Profile{}, err
	}
	release, err := in.github.LatestRelease(ctx, owner, repo)
	if err != nil {
		return library.Profile{}, err
	}
	asset, err := PickAsset(release, m.Source.Asset)
	if err != nil {
		return library.Profile{}, err
	}
	path, err := in.github.Download(ctx, owner, repo, release.Tag, asset, nil)
	if err != nil {
		return library.Profile{}, err
	}
	pkg, err := InspectFile(path)
	if err != nil {
		return library.Profile{}, err
	}
	// Keep the mod's identity; only the version moves.
	pkg.Author, pkg.Name, pkg.Version = m.Author, m.Name, NormalizeVersion(release.Tag)
	return in.InstallGitHub(ctx, gameID, profileID, pkg, owner, repo, release.Tag, asset.Name, false, onProgress)
}

// PickAsset chooses the release file to install: the named one if present,
// otherwise the only .zip or .dll of the release.
func PickAsset(release github.Release, preferred string) (github.Asset, error) {
	var mods []github.Asset
	for _, a := range release.Assets {
		if a.Name == preferred && preferred != "" {
			return a, nil
		}
		if ext := strings.ToLower(filepath.Ext(a.Name)); ext == ".zip" || ext == ".dll" {
			mods = append(mods, a)
		}
	}
	if len(mods) == 1 {
		return mods[0], nil
	}
	names := make([]string, len(mods))
	for i, a := range mods {
		names[i] = a.Name
	}
	slices.Sort(names)
	return github.Asset{}, fmt.Errorf("release %s has %d mod files (%s); install the update manually", release.Tag, len(mods), strings.Join(names, ", "))
}
