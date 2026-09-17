package app

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"

	"bepinexmodmanager/internal/library"
	"bepinexmodmanager/internal/modinstall"
	"bepinexmodmanager/internal/thunderstore"
)

// InstallProgressEvent carries modinstall.Progress to the frontend.
const InstallProgressEvent = "install:progress"

type InstallService struct {
	installer *modinstall.Installer
}

func NewInstallService(installer *modinstall.Installer) *InstallService {
	return &InstallService{installer: installer}
}

// InstallPackage installs a Thunderstore package version with its dependencies
// into a profile, emitting InstallProgressEvent along the way.
func (s *InstallService) InstallPackage(ctx context.Context, gameID, profileID, namespace, name, version string) (library.Profile, error) {
	ref := thunderstore.PackageRef{Namespace: namespace, Name: name, Version: version}
	return s.installer.Install(ctx, gameID, profileID, ref, func(p modinstall.Progress) {
		if app := application.Get(); app != nil {
			app.Event.Emit(InstallProgressEvent, p)
		}
	})
}

func (s *InstallService) UninstallMod(gameID, profileID, modID string) (library.Profile, error) {
	return s.installer.Uninstall(gameID, profileID, modID)
}

// SetModEnabled enables or disables a mod; dependants follow automatically.
func (s *InstallService) SetModEnabled(gameID, profileID, modID string, enabled bool) (library.Profile, error) {
	return s.installer.SetEnabled(gameID, profileID, modID, enabled)
}

// OpenProfile returns a profile with mod states brought up to date.
func (s *InstallService) OpenProfile(gameID, profileID string) (library.Profile, error) {
	return s.installer.Refresh(gameID, profileID)
}
