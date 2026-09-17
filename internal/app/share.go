package app

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"

	"bepinexmodmanager/internal/profileshare"
)

// ImportProgressEvent carries profileshare.ImportProgress to the frontend.
const ImportProgressEvent = "import:progress"

// ShareService exports and imports profiles in r2modman's format.
type ShareService struct {
	sharer *profileshare.Sharer
}

func NewShareService(sharer *profileshare.Sharer) *ShareService {
	return &ShareService{sharer: sharer}
}

func (s *ShareService) ExportFile(gameID, profileID, dest string) error {
	return s.sharer.ExportFile(gameID, profileID, dest)
}

// ExportCode uploads the profile to Thunderstore and returns a code anyone can import.
func (s *ShareService) ExportCode(ctx context.Context, gameID, profileID string) (string, error) {
	return s.sharer.ExportCode(ctx, gameID, profileID)
}

func (s *ShareService) PreviewImport(ctx context.Context, src profileshare.Source) (profileshare.Preview, error) {
	return s.sharer.Preview(ctx, src)
}

// Import creates a profile from a previewed archive, emitting
// ImportProgressEvent and InstallProgressEvent along the way.
func (s *ShareService) Import(ctx context.Context, gameID, archive, name string) (profileshare.ImportResult, error) {
	return s.sharer.Import(ctx, gameID, archive, name, func(p profileshare.ImportProgress) {
		if app := application.Get(); app != nil {
			app.Event.Emit(ImportProgressEvent, p)
		}
	}, emitProgress)
}
