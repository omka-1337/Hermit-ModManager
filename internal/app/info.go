// Package app holds application-level metadata and services exposed to the frontend.
package app

import "runtime"

const Name = "BepInEx Mod Manager"

// Version is overridden at build time via -ldflags "-X bepinexmodmanager/internal/app.Version=...".
var Version = "0.1.0-dev"

type AppInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

type InfoService struct{}

func NewInfoService() *InfoService {
	return &InfoService{}
}

func (s *InfoService) GetInfo() AppInfo {
	return AppInfo{
		Name:    Name,
		Version: Version,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
}
