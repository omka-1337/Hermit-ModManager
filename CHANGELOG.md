# Changelog

## 0.2.0 — 2026-09-18

### Added

- Steam Deck layout: a console-style bar of games along the top, switched with LB/RB, full screen and without a title
  bar.
- Controller navigation: the d-pad moves the focus, A selects, B steps back and finally leaves Hermit. Dialogs and
  confirmations answer to the controller as well, and LT/RT switch the tabs of a profile.
- Card view in the mod browser, with a cards/list switch that is remembered between runs.
- Update progress: a spinner on the mod being updated, a green tick when it is done, and a visible result of a check.
- Update check for Hermit itself: About tells when a newer release exists and links to it.
- Licensed under the GNU GPL v3 or later, with the licenses of every dependency listed in
  [THIRD-PARTY.md](THIRD-PARTY.md).
- Screenshots and install instructions in the README.

### Changed

- The running game and the Steam launch options are polled, so their state no longer sticks until the page is
  reopened.
- Adding a game in the Steam Deck layout is a page of its own instead of a dialog.

### Fixed

- Flicker when switching between games: icons are painted from cache and the page keeps its shape while it loads.
- Entering a game page focused the rename pencil instead of the active profile.
- The selection ring in the game bar was clipped at the top and bottom.
- Steam's `localconfig.vdf` was reparsed on every poll; it is now cached until the file changes.

## 0.1.0 — 2026-09-17

First release, as an AppImage.

### Added

- Games found in Steam libraries, with the runtime (Proton or native) and the Unity backend detected automatically.
- Profiles per game: mods live in the profile and reach the game through symlinks, so the game folder stays untouched.
- Thunderstore browser with dependency resolution, conflict detection, updates and modpacks, where a modpack becomes a
  profile of its own.
- Mods from local `.zip` and `.dll` files and from GitHub releases, with BepInEx plugin metadata read out of the
  assemblies.
- Enabling and disabling mods, with a mod force-disabled while its dependencies are missing.
- Config file editor for installed mods.
- Launching from Hermit or straight from Steam through launch options, so mods load without the manager running.
- Profile export and import compatible with r2modman: `.r2z` files and Thunderstore profile codes.
- A custom window frame instead of the desktop's own decorations.
