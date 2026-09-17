package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParseRepo(t *testing.T) {
	ok := map[string][2]string{
		"BepInEx/BepInEx":                                {"BepInEx", "BepInEx"},
		"https://github.com/owner/my.mod.git":            {"owner", "my.mod"},
		"github.com/owner/repo/releases/tag/v1.0":        {"owner", "repo"},
		" https://www.github.com/owner/repo?tab=readme ": {"owner", "repo"},
	}
	for in, want := range ok {
		o, r, err := ParseRepo(in)
		if err != nil || o != want[0] || r != want[1] {
			t.Errorf("ParseRepo(%q) = %q %q %v", in, o, r, err)
		}
	}
	for _, bad := range []string{"", "owner", "a/b/c", "../x", "https://gitlab.com/a/b", "a b/c"} {
		if _, _, err := ParseRepo(bad); err == nil {
			t.Errorf("ParseRepo(%q) should fail", bad)
		}
	}
}

func TestReleasesAndDownload(t *testing.T) {
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/releases", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"tag_name": "v2.0.0-beta", "prerelease": true, "assets": [{"name": "Mod.dll", "size": 3, "browser_download_url": "https://github.com/o/r/releases/download/v2.0.0-beta/Mod.dll"}]},
			{"tag_name": "v1.1.0", "draft": true, "assets": [{"name": "Mod.dll", "size": 3}]},
			{"tag_name": "v1.0.0", "assets": [{"name": "Mod.dll", "size": 3, "browser_download_url": "https://github.com/o/r/releases/download/v1.0.0/Mod.dll"}]},
			{"tag_name": "v0.9.0", "assets": []}
		]`))
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "test", t.TempDir())
	ctx := context.Background()
	releases, err := c.Releases(ctx, "o", "r")
	if err != nil || len(releases) != 2 {
		t.Fatalf("releases: %+v %v", releases, err)
	}
	latest, err := c.LatestRelease(ctx, "o", "r")
	if err != nil || latest.Tag != "v1.0.0" {
		t.Errorf("latest: %+v %v", latest, err)
	}
	if _, err := c.Releases(ctx, "o", "missing"); err == nil {
		t.Error("missing repo must fail")
	}

	// Downloads go to github.com; point the client's HTTP at the test server.
	c.http = &http.Client{Transport: rewriteTransport{target: srv.URL}}
	mux.HandleFunc("/o/r/releases/download/v1.0.0/Mod.dll", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("dll"))
	})
	path, err := c.Download(ctx, "o", "r", "v1.0.0", latest.Assets[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); string(data) != "dll" {
		t.Errorf("downloaded %q", data)
	}
	if _, err := c.Download(ctx, "o", "r", "v1.0.0", Asset{Name: "../evil.dll", URL: "https://github.com/x"}, nil); err == nil {
		t.Error("asset names with paths must be rejected")
	}
}

type rewriteTransport struct{ target string }

func (rt rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	u := *r.URL
	target, _ := http.NewRequest(http.MethodGet, rt.target, nil)
	u.Scheme, u.Host = target.URL.Scheme, target.URL.Host
	r2 := r.Clone(r.Context())
	r2.URL = &u
	return http.DefaultTransport.RoundTrip(r2)
}
