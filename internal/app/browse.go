package app

import (
	"context"
	"errors"

	"bepinexmodmanager/internal/library"
	"bepinexmodmanager/internal/thunderstore"
)

var ErrNoCommunity = errors.New("this game is not available on Thunderstore")

// BrowseService lets the frontend browse Thunderstore packages for a game.
type BrowseService struct {
	lib *library.Library
	ts  *thunderstore.Client
}

func NewBrowseService(lib *library.Library, ts *thunderstore.Client) *BrowseService {
	return &BrowseService{lib: lib, ts: ts}
}

// GetCommunity returns the Thunderstore community of a game, or ErrNoCommunity.
func (s *BrowseService) GetCommunity(ctx context.Context, gameID string) (thunderstore.Community, error) {
	game, err := s.lib.GetGame(gameID)
	if err != nil {
		return thunderstore.Community{}, err
	}
	schema, err := s.ts.Schema(ctx)
	if err != nil {
		return thunderstore.Community{}, err
	}
	community, ok := schema.FindCommunity(game.SteamAppID, game.Executable)
	if !ok {
		return thunderstore.Community{}, ErrNoCommunity
	}
	return community, nil
}

func (s *BrowseService) GetFilters(ctx context.Context, community string) (thunderstore.Filters, error) {
	return s.ts.Filters(ctx, community)
}

func (s *BrowseService) ListPackages(ctx context.Context, community string, opts thunderstore.ListOptions) (thunderstore.PackageList, error) {
	return s.ts.ListPackages(ctx, community, opts)
}

func (s *BrowseService) GetPackage(ctx context.Context, community, namespace, name string) (thunderstore.PackageDetail, error) {
	return s.ts.Package(ctx, community, namespace, name)
}

func (s *BrowseService) GetReadme(ctx context.Context, namespace, name, version string) (string, error) {
	return s.ts.Readme(ctx, namespace, name, version)
}
