package thunderstore

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Manifest is the manifest.json at the root of every Thunderstore package.
type Manifest struct {
	Namespace     string   `json:"namespace"`
	Name          string   `json:"name"`
	VersionNumber string   `json:"version_number"`
	Description   string   `json:"description"`
	WebsiteURL    string   `json:"website_url"`
	Dependencies  []string `json:"dependencies"`
}

func ReadManifest(zipPath string) (Manifest, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return Manifest{}, err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if !strings.EqualFold(f.Name, "manifest.json") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return Manifest{}, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return Manifest{}, err
		}
		var m Manifest
		// Many manifests are saved with a UTF-8 BOM.
		if err := json.Unmarshal(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), &m); err != nil {
			return Manifest{}, fmt.Errorf("parse manifest.json: %w", err)
		}
		return m, nil
	}
	return Manifest{}, fmt.Errorf("manifest.json not found in %s", zipPath)
}
