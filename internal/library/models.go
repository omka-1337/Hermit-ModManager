package library

type Runtime string

const (
	RuntimeUnknown Runtime = "unknown"
	RuntimeNative  Runtime = "native" // Linux build, BepInEx via run_bepinex.sh
	RuntimeProton  Runtime = "proton" // Windows build run through Proton/Wine
)

// Game is stored in games/<id>/game.json. It holds everything machine-specific
// (paths, runtime); profiles must stay portable.
type Game struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Path          string  `json:"path"`
	Runtime       Runtime `json:"runtime"`
	ActiveProfile string  `json:"activeProfile"`
}

const ProfileSchemaVersion = 1

// Profile is stored in profiles/<id>/profile.json and is meant to be shareable:
// no absolute paths or machine-specific data.
type Profile struct {
	SchemaVersion int    `json:"schemaVersion"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Mods          []Mod  `json:"mods"`
}

type Mod struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Version string    `json:"version"`
	Enabled bool      `json:"enabled"`
	Source  ModSource `json:"source"`
	// Files are paths relative to the profile's BepInEx directory.
	Files []string `json:"files"`
}

type ModSourceType string

const (
	SourceLocal        ModSourceType = "local"
	SourceThunderstore ModSourceType = "thunderstore"
	SourceURL          ModSourceType = "url"
)

// ModSource describes where to re-fetch a mod when importing a profile.
type ModSource struct {
	Type   ModSourceType `json:"type"`
	URL    string        `json:"url,omitempty"`
	SHA256 string        `json:"sha256,omitempty"`
}
