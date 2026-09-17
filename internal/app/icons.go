package app

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"

	"bepinexmodmanager/internal/steam"
)

// IconService provides game icons as data URLs, so the webview can show local
// Steam cache files.
type IconService struct {
	steamRoots []string
}

func NewIconService(steamRoots []string) *IconService {
	return &IconService{steamRoots: steamRoots}
}

var iconTypes = map[string]string{".ico": "image/x-icon", ".jpg": "image/jpeg", ".png": "image/png"}

// GetSteamIcon returns the icon of a Steam app as a data URL, or "" if none is cached.
func (s *IconService) GetSteamIcon(appID string) string {
	path := steam.IconPath(s.steamRoots, appID)
	mime := iconTypes[strings.ToLower(filepath.Ext(path))]
	if path == "" || mime == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 2<<20 {
		return ""
	}
	if mime == "image/x-icon" {
		if png := largestPNGInICO(data); png != nil {
			data, mime = png, "image/png"
		}
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// largestPNGInICO returns the largest PNG-encoded image of an .ico file, or
// nil if it has none (older icons store bitmaps).
func largestPNGInICO(ico []byte) []byte {
	if len(ico) < 6 || binary.LittleEndian.Uint16(ico[2:4]) != 1 {
		return nil
	}
	count := int(binary.LittleEndian.Uint16(ico[4:6]))
	var best []byte
	bestSize := 0
	for i := 0; i < count; i++ {
		entry := 6 + 16*i
		if entry+16 > len(ico) {
			break
		}
		width := int(ico[entry])
		if width == 0 {
			width = 256
		}
		size := int(binary.LittleEndian.Uint32(ico[entry+8:]))
		offset := int(binary.LittleEndian.Uint32(ico[entry+12:]))
		if offset < 0 || size <= 0 || offset+size > len(ico) {
			continue
		}
		img := ico[offset : offset+size]
		if width > bestSize && bytes.HasPrefix(img, []byte("\x89PNG\r\n\x1a\n")) {
			best, bestSize = img, width
		}
	}
	return best
}
