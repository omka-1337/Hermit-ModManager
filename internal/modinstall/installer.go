package modinstall

import (
	"context"
	"errors"
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

// Repository is where packages come from (the Thunderstore client).
type Repository interface {
	DownloadPackage(ctx context.Context, ref thunderstore.PackageRef, progress func(done, total int64)) (thunderstore.Archive, error)
	Versions(ctx context.Context, namespace, name string) ([]thunderstore.Version, error)
}

// Options control how a package is installed.
type Options struct {
	// ReplaceConflicts uninstalls installed mods that conflict with the
	// packages being installed.
	ReplaceConflicts bool `json:"replaceConflicts"`
	// Modpack installs exact dependency versions, so everyone installing the
	// same modpack gets the same set of mods.
	Modpack bool `json:"modpack"`
	// Pinned maps mod ids to the versions to use whenever they appear as a
	// dependency, e.g. the mod list of an imported profile.
	Pinned map[string]string `json:"-"`
}

// RulesFunc returns the install rules of a game.
type RulesFunc func(ctx context.Context, game library.Game) Rules

type Installer struct {
	lib   *library.Library
	repo  Repository
	rules RulesFunc

	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// NewInstaller creates an installer; rules may be nil to use DefaultRules.
func NewInstaller(lib *library.Library, repo Repository, rules RulesFunc) *Installer {
	if rules == nil {
		rules = func(context.Context, library.Game) Rules { return DefaultRules() }
	}
	return &Installer{lib: lib, repo: repo, rules: rules, locks: map[string]*sync.Mutex{}}
}

func (in *Installer) rulesFor(ctx context.Context, gameID string) (Rules, error) {
	game, err := in.lib.GetGame(gameID)
	if err != nil {
		return Rules{}, err
	}
	return in.rules(ctx, game), nil
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
	rules    Rules
	files    []string
}

type Action string

const (
	ActionInstall   Action = "install"
	ActionUpdate    Action = "update"
	ActionReinstall Action = "reinstall"
)

type PlannedPackage struct {
	// Package is "<author>-<name>-<version>".
	Package string `json:"package"`
	Action  Action `json:"action"`
}

type ConflictReason string

const (
	// ConflictFiles means both packages install the same files.
	ConflictFiles ConflictReason = "files"
	// ConflictLoader means both packages are BepInEx loader packs; only one can be installed.
	ConflictLoader ConflictReason = "loader"
)

type Conflict struct {
	// Package is the planned package, "<author>-<name>-<version>".
	Package string `json:"package"`
	// ModID is the conflicting mod: installed, or another planned package.
	ModID     string         `json:"modId"`
	ModName   string         `json:"modName"`
	Installed bool           `json:"installed"`
	Reason    ConflictReason `json:"reason"`
	// Blocking conflicts cannot be solved by replacing: the other side is
	// itself part of, or required by, the packages being installed.
	Blocking bool `json:"blocking"`
	// Files lists some of the shared files.
	Files []string `json:"files"`
}

type InstallPlan struct {
	Packages  []PlannedPackage `json:"packages"`
	Conflicts []Conflict       `json:"conflicts"`
}

var ErrConflicts = errors.New("conflicts with installed mods")

// PlanInstall resolves what installing a package would do, downloading archives
// as needed, without changing the profile.
func (in *Installer) PlanInstall(ctx context.Context, gameID, profileID string, ref thunderstore.PackageRef, opts Options, onProgress func(Progress)) (InstallPlan, error) {
	if err := ref.Validate(); err != nil {
		return InstallPlan{}, err
	}
	lock := in.profileLock(gameID, profileID)
	lock.Lock()
	defer lock.Unlock()

	profile, err := in.lib.GetProfile(gameID, profileID)
	if err != nil {
		return InstallPlan{}, err
	}
	rules, err := in.rulesFor(ctx, gameID)
	if err != nil {
		return InstallPlan{}, err
	}
	plan, err := in.resolve(ctx, profile, ref, opts, rules, in.progressFunc(gameID, profileID, ref, onProgress))
	if err != nil {
		return InstallPlan{}, err
	}
	return describePlan(profile, plan), nil
}

// Install installs a package version together with its dependencies, which
// are chosen as described at resolve.
//
// If planned packages conflict with installed mods, Install fails with
// ErrConflicts unless opts.ReplaceConflicts is set, in which case the
// conflicting mods are uninstalled first. Conflicts between planned packages
// always fail.
func (in *Installer) Install(ctx context.Context, gameID, profileID string, ref thunderstore.PackageRef, opts Options, onProgress func(Progress)) (library.Profile, error) {
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
	rules, err := in.rulesFor(ctx, gameID)
	if err != nil {
		return library.Profile{}, err
	}
	report := in.progressFunc(gameID, profileID, ref, onProgress)
	plan, err := in.resolve(ctx, profile, ref, opts, rules, report)
	if err != nil {
		return library.Profile{}, err
	}

	var replace []string
	for _, c := range findConflicts(profile, plan) {
		if c.Blocking {
			return library.Profile{}, fmt.Errorf("%s and %s cannot be installed together (%s)", c.Package, c.ModID, c.Reason)
		}
		if !opts.ReplaceConflicts {
			return library.Profile{}, fmt.Errorf("%s: %w: %s", c.Package, ErrConflicts, c.ModName)
		}
		if !slices.Contains(replace, c.ModID) {
			replace = append(replace, c.ModID)
		}
	}
	if len(replace) > 0 {
		_, err := in.updateProfile(gameID, profileID, func(p *library.Profile) (error, error) {
			var errs []error
			for _, id := range replace {
				if i := slices.IndexFunc(p.Mods, func(m library.Mod) bool { return m.ID == id }); i >= 0 {
					errs = append(errs, removeModFiles(profileDir, p.Mods[i], p.Mods))
					p.Mods = slices.Delete(p.Mods, i, i+1)
				}
			}
			return errors.Join(append(errs, Sync(profileDir, p))...), nil
		})
		if err != nil {
			return library.Profile{}, err
		}
	}

	for _, p := range plan {
		if err := ctx.Err(); err != nil {
			return library.Profile{}, err
		}
		report(p.ref, StageInstall, 0, 0)
		if _, err := in.installOne(gameID, profileID, profileDir, p); err != nil {
			return library.Profile{}, err
		}
	}
	return in.lib.GetProfile(gameID, profileID)
}

func (in *Installer) progressFunc(gameID, profileID string, target thunderstore.PackageRef, onProgress func(Progress)) func(thunderstore.PackageRef, Stage, int64, int64) {
	return func(pkg thunderstore.PackageRef, stage Stage, done, total int64) {
		if onProgress != nil {
			onProgress(Progress{GameID: gameID, ProfileID: profileID, Target: target.ID(), Package: pkg.String(), Stage: stage, Done: done, Total: total})
		}
	}
}

// resolve downloads the package and its dependency tree and returns the
// packages to install, dependencies first. The requested package is always
// included, in the requested version. Dependencies follow r2modman:
//
//   - for a modpack, the exact versions from the manifests are installed;
//   - otherwise a missing dependency is installed in its latest version and an
//     installed one is left alone, unless it is older than required, in which
//     case it is updated to the latest version (r2modman keeps it as is).
func (in *Installer) resolve(ctx context.Context, profile library.Profile, ref thunderstore.PackageRef, opts Options, rules Rules, report func(thunderstore.PackageRef, Stage, int64, int64)) ([]plannedPackage, error) {
	installed := map[string]string{}
	for _, m := range profile.Mods {
		installed[m.ID] = m.Version
	}

	var plan []plannedPackage
	planned := map[string]int{}
	var visit func(r thunderstore.PackageRef, root bool) error
	visit = func(r thunderstore.PackageRef, root bool) error {
		if i, ok := planned[r.ID()]; ok {
			if i < 0 || CompareVersions(plan[i].ref.Version, r.Version) >= 0 {
				return nil // being resolved (cycle) or already planned at a sufficient version
			}
		}
		if pinned, ok := opts.Pinned[r.ID()]; ok && !root {
			if installed[r.ID()] == pinned {
				return nil
			}
			r.Version = pinned
		} else if !root {
			v, ok := installed[r.ID()]
			switch {
			case opts.Modpack && v == r.Version:
				return nil
			case opts.Modpack:
			case ok && CompareVersions(v, r.Version) >= 0:
				return nil
			default:
				r.Version = in.latestVersion(ctx, r)
			}
		}
		planned[r.ID()] = -1
		archive, err := in.repo.DownloadPackage(ctx, r, func(done, total int64) {
			report(r, StageDownload, done, total)
		})
		if err != nil {
			return err
		}
		manifest, err := thunderstore.ReadManifest(archive.Path)
		if err != nil {
			return fmt.Errorf("%s: %w", r, err)
		}
		files, err := PlanFiles(archive.Path, r.ID(), rules)
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
		plan = append(plan, plannedPackage{ref: r, archive: archive, manifest: manifest, rules: rules, files: files})
		planned[r.ID()] = len(plan) - 1
		return nil
	}
	if err := visit(ref, true); err != nil {
		return nil, err
	}

	// Drop entries superseded by a newer version planned later.
	var result []plannedPackage
	for i, p := range plan {
		if planned[p.ref.ID()] == i {
			result = append(result, p)
		}
	}
	return result, nil
}

// Update is an installed mod with a newer version available.
type Update struct {
	ModID   string `json:"modId"`
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

// CheckUpdates returns the Thunderstore mods of a profile that have a newer
// version. Mods whose versions cannot be fetched are skipped.
func (in *Installer) CheckUpdates(ctx context.Context, gameID, profileID string) ([]Update, error) {
	profile, err := in.lib.GetProfile(gameID, profileID)
	if err != nil {
		return nil, err
	}
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		updates = []Update{}
		limit   = make(chan struct{}, 6)
	)
	for _, m := range profile.Mods {
		if m.Source.Type != library.SourceThunderstore {
			continue
		}
		ref := thunderstore.PackageRef{Namespace: m.Author, Name: m.Name, Version: m.Version}
		wg.Add(1)
		go func() {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			if latest := in.latestVersion(ctx, ref); CompareVersions(latest, ref.Version) > 0 {
				mu.Lock()
				updates = append(updates, Update{ModID: ref.ID(), Current: ref.Version, Latest: latest})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	slices.SortFunc(updates, func(a, b Update) int { return strings.Compare(a.ModID, b.ModID) })
	return updates, nil
}

type UpdateResult struct {
	Profile library.Profile `json:"profile"`
	Updated []string        `json:"updated"`
	// Failed maps mod ids to the reason they could not be updated.
	Failed map[string]string `json:"failed"`
}

// UpdateAll updates every outdated mod of a profile to its latest version, the
// way r2modman's "Update all" does; new dependencies get their latest version
// too. A mod that fails, e.g. because of a conflict, does not stop the others.
func (in *Installer) UpdateAll(ctx context.Context, gameID, profileID string, onProgress func(Progress)) (UpdateResult, error) {
	updates, err := in.CheckUpdates(ctx, gameID, profileID)
	if err != nil {
		return UpdateResult{}, err
	}
	result := UpdateResult{Updated: []string{}, Failed: map[string]string{}}
	for _, u := range updates {
		// An earlier update may already have pulled this one in as a dependency.
		if current, err := in.lib.GetProfile(gameID, profileID); err == nil {
			if i := slices.IndexFunc(current.Mods, func(m library.Mod) bool { return m.ID == u.ModID }); i >= 0 &&
				CompareVersions(current.Mods[i].Version, u.Latest) >= 0 {
				result.Updated = append(result.Updated, u.ModID)
				continue
			}
		}
		ref, err := thunderstore.ParseDependency(u.ModID + "-" + u.Latest)
		if err == nil {
			_, err = in.Install(ctx, gameID, profileID, ref, Options{}, onProgress)
		}
		if err != nil {
			if ctx.Err() != nil {
				return UpdateResult{}, ctx.Err()
			}
			result.Failed[u.ModID] = err.Error()
			continue
		}
		result.Updated = append(result.Updated, u.ModID)
	}
	result.Profile, err = in.lib.GetProfile(gameID, profileID)
	return result, err
}

// latestVersion returns the newest published version of a package, or the
// given version if the list cannot be fetched.
func (in *Installer) latestVersion(ctx context.Context, r thunderstore.PackageRef) string {
	versions, err := in.repo.Versions(ctx, r.Namespace, r.Name)
	if err != nil {
		return r.Version
	}
	latest := r.Version
	for _, v := range versions {
		if CompareVersions(v.VersionNumber, latest) > 0 {
			latest = v.VersionNumber
		}
	}
	return latest
}

func describePlan(profile library.Profile, plan []plannedPackage) InstallPlan {
	installed := map[string]string{}
	for _, m := range profile.Mods {
		installed[m.ID] = m.Version
	}
	result := InstallPlan{Packages: []PlannedPackage{}, Conflicts: findConflicts(profile, plan)}
	for _, p := range plan {
		action := ActionInstall
		if v, ok := installed[p.ref.ID()]; ok {
			action = ActionUpdate
			if v == p.ref.Version {
				action = ActionReinstall
			}
		}
		result.Packages = append(result.Packages, PlannedPackage{Package: p.ref.String(), Action: action})
	}
	return result
}

// findConflicts checks planned packages against installed mods (other than
// the ones they replace) and against each other.
func findConflicts(profile library.Profile, plan []plannedPackage) []Conflict {
	type owner struct {
		id, name  string
		installed bool
		files     []string
	}
	replaced := map[string]bool{}
	required := map[string]bool{}
	for _, p := range plan {
		replaced[p.ref.ID()] = true
		for _, dep := range p.manifest.Dependencies {
			if r, err := thunderstore.ParseDependency(dep); err == nil {
				required[r.ID()] = true
			}
		}
	}
	var owners []owner
	for _, m := range profile.Mods {
		if !replaced[m.ID] {
			owners = append(owners, owner{m.ID, m.Name, true, m.Files})
		}
	}

	conflicts := []Conflict{}
	for _, p := range plan {
		loader := IsLoader(p.files)
		for _, o := range owners {
			var shared []string
			for _, f := range p.files {
				if slices.Contains(o.files, f) {
					shared = append(shared, f)
				}
			}
			c := Conflict{
				Package:   p.ref.String(),
				ModID:     o.id,
				ModName:   o.name,
				Installed: o.installed,
				Reason:    ConflictFiles,
				Blocking:  !o.installed || required[o.id],
			}
			switch {
			case loader && IsLoader(o.files):
				c.Reason = ConflictLoader
			case len(shared) == 0:
				continue
			}
			c.Files = shared[:min(len(shared), 5)]
			if c.Files == nil {
				c.Files = []string{}
			}
			conflicts = append(conflicts, c)
		}
		owners = append(owners, owner{p.ref.ID(), p.ref.Name, false, p.files})
	}
	return conflicts
}

var ErrUnmetDependencies = errors.New("missing or disabled dependencies")

// updateProfile applies fn and always saves the profile, so profile.json keeps
// matching the files on disk even when a file operation fails midway. fn
// returns validation errors (profile not saved) and file errors separately.
func (in *Installer) updateProfile(gameID, profileID string, fn func(*library.Profile) (fileErr, validationErr error)) (library.Profile, error) {
	var fileErr error
	prof, err := in.lib.UpdateProfile(gameID, profileID, func(p *library.Profile) error {
		var validationErr error
		fileErr, validationErr = fn(p)
		return validationErr
	})
	if err != nil {
		return library.Profile{}, err
	}
	return prof, fileErr
}

func (in *Installer) installOne(gameID, profileID, profileDir string, p plannedPackage) (library.Profile, error) {
	return in.updateProfile(gameID, profileID, func(prof *library.Profile) (error, error) {
		enabled := true
		if i := slices.IndexFunc(prof.Mods, func(m library.Mod) bool { return m.ID == p.ref.ID() }); i >= 0 {
			old := prof.Mods[i]
			enabled = old.Enabled // an update keeps the user's choice
			if err := removeModFiles(profileDir, old, prof.Mods); err != nil {
				return fmt.Errorf("remove old %s: %w", old.ID, err), nil
			}
			prof.Mods = slices.Delete(prof.Mods, i, i+1)
		}

		files, err := Extract(p.archive.Path, profileDir, p.ref.ID(), p.rules)
		if err != nil {
			// The old version is already gone, so the profile is saved without it.
			return errors.Join(fmt.Errorf("install %s: %w", p.ref, err), Sync(profileDir, prof)), nil
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
			Enabled: enabled,
			Active:  true, // Extract puts files in the active location; Sync fixes it up
			Source: library.ModSource{
				Type:   library.SourceThunderstore,
				URL:    p.archive.URL,
				SHA256: p.archive.SHA256,
			},
			Dependencies: deps,
			Files:        files,
		})
		slices.SortFunc(prof.Mods, func(a, b library.Mod) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
		return Sync(profileDir, prof), nil
	})
}

// Uninstall removes a mod's files and its entry from the profile. Dependencies
// are left installed; dependants become inactive.
func (in *Installer) Uninstall(gameID, profileID, modID string) (library.Profile, error) {
	lock := in.profileLock(gameID, profileID)
	lock.Lock()
	defer lock.Unlock()

	profileDir, err := in.lib.ProfileDir(gameID, profileID)
	if err != nil {
		return library.Profile{}, err
	}
	return in.updateProfile(gameID, profileID, func(p *library.Profile) (error, error) {
		i := slices.IndexFunc(p.Mods, func(m library.Mod) bool { return m.ID == modID })
		if i < 0 {
			return nil, fmt.Errorf("mod %q: %w", modID, library.ErrNotFound)
		}
		removeErr := removeModFiles(profileDir, p.Mods[i], p.Mods)
		p.Mods = slices.Delete(p.Mods, i, i+1)
		return errors.Join(removeErr, Sync(profileDir, p)), nil
	})
}

// Refresh recomputes mod states, e.g. for profiles saved by older versions or
// changed outside the manager.
func (in *Installer) Refresh(gameID, profileID string) (library.Profile, error) {
	lock := in.profileLock(gameID, profileID)
	lock.Lock()
	defer lock.Unlock()

	profileDir, err := in.lib.ProfileDir(gameID, profileID)
	if err != nil {
		return library.Profile{}, err
	}
	return in.updateProfile(gameID, profileID, func(p *library.Profile) (error, error) {
		return Sync(profileDir, p), nil
	})
}

// SetEnabled enables or disables a mod. Enabling fails with
// ErrUnmetDependencies while a dependency is missing or disabled.
func (in *Installer) SetEnabled(gameID, profileID, modID string, enabled bool) (library.Profile, error) {
	lock := in.profileLock(gameID, profileID)
	lock.Lock()
	defer lock.Unlock()

	profileDir, err := in.lib.ProfileDir(gameID, profileID)
	if err != nil {
		return library.Profile{}, err
	}
	return in.updateProfile(gameID, profileID, func(p *library.Profile) (error, error) {
		i := slices.IndexFunc(p.Mods, func(m library.Mod) bool { return m.ID == modID })
		if i < 0 {
			return nil, fmt.Errorf("mod %q: %w", modID, library.ErrNotFound)
		}
		if enabled && len(p.Mods[i].UnmetDependencies) > 0 {
			return nil, fmt.Errorf("cannot enable %s: %w: %s", p.Mods[i].Name, ErrUnmetDependencies, strings.Join(p.Mods[i].UnmetDependencies, ", "))
		}
		p.Mods[i].Enabled = enabled
		return Sync(profileDir, p), nil
	})
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
