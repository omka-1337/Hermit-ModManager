package app

import (
	"context"
	"errors"

	"hermit/internal/library"
	"hermit/internal/settings"
	"hermit/internal/thunderstore"
)

var ErrNoCommunity = errors.New("this game is not available on Thunderstore")

// BrowseService lets the frontend browse Thunderstore packages for a game.
type BrowseService struct {
	lib      *library.Library
	ts       *thunderstore.Client
	settings *settings.Store
}

func NewBrowseService(lib *library.Library, ts *thunderstore.Client, settings *settings.Store) *BrowseService {
	return &BrowseService{lib: lib, ts: ts, settings: settings}
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
	st, err := s.settings.Get()
	if err != nil {
		return thunderstore.PackageList{}, err
	}
	opts.IncludeNSFW = st.AllowNSFW
	return s.ts.ListPackages(ctx, community, opts)
}

func (s *BrowseService) GetPackage(ctx context.Context, community, namespace, name string) (thunderstore.PackageDetail, error) {
	return s.ts.Package(ctx, community, namespace, name)
}

func (s *BrowseService) GetReadme(ctx context.Context, namespace, name, version string) (string, error) {
	return s.ts.Readme(ctx, namespace, name, version)
}
