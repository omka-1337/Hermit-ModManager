package app

import (
	"context"
	"errors"
	"testing"

	"hermit/internal/github"
)

type fakeReleases struct {
	release github.Release
	err     error
}

func (f fakeReleases) LatestRelease(context.Context, string, string) (github.Release, error) {
	return f.release, f.err
}

func TestCheckUpdate(t *testing.T) {
	old := Version
	Version = "0.1.0"
	t.Cleanup(func() { Version = old })

	tests := []struct {
		name      string
		tag       string
		wantLates string
		available bool
	}{
		{name: "newer release", tag: "v0.2.0", wantLates: "0.2.0", available: true},
		{name: "same version", tag: "v0.1.0", wantLates: "0.1.0"},
		{name: "older release", tag: "v0.0.9", wantLates: "0.0.9"},
		{name: "tag without v", tag: "0.3.1", wantLates: "0.3.1", available: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewInfoService(fakeReleases{release: github.Release{Tag: tc.tag}})
			got, err := s.CheckUpdate(context.Background())
			if err != nil {
				t.Fatalf("CheckUpdate: %v", err)
			}
			if got.Latest != tc.wantLates {
				t.Errorf("latest = %q, want %q", got.Latest, tc.wantLates)
			}
			if got.Available != tc.available {
				t.Errorf("available = %v, want %v", got.Available, tc.available)
			}
			if got.Current != "0.1.0" {
				t.Errorf("current = %q, want 0.1.0", got.Current)
			}
			if want := RepoURL + "/releases/tag/" + tc.tag; got.URL != want {
				t.Errorf("url = %q, want %q", got.URL, want)
			}
		})
	}
}

func TestCheckUpdateError(t *testing.T) {
	s := NewInfoService(fakeReleases{err: errors.New("offline")})
	if _, err := s.CheckUpdate(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
}
