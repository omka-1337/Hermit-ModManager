package thunderstore

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const schemaJSON = `{
  "games": {
    "lethal-company": {
      "label": "lethal-company",
      "distributions": [{"platform": "steam", "identifier": "1966720"}],
      "r2modman": [{"exeNames": ["Lethal Company.exe"], "packageLoader": "bepinex",
        "distributions": [{"platform": "steam", "identifier": "1966720"}]}],
      "thunderstore": {"displayName": "Lethal Company"}
    },
    "melon-game": {
      "label": "melon-game",
      "distributions": [{"platform": "steam", "identifier": "42"}],
      "r2modman": [{"exeNames": ["Melon.exe"], "packageLoader": "melonloader"}],
      "thunderstore": {"displayName": "Melon Game"}
    }
  }
}`

func newTestServer(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	schemaHits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/experimental/schema/dev/latest/", func(w http.ResponseWriter, r *http.Request) {
		schemaHits++
		w.Write([]byte(schemaJSON))
	})
	mux.HandleFunc("/api/cyberstorm/listing/lethal-company/{$}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "more" || r.URL.Query().Get("ordering") != "most-downloaded" || r.URL.Query().Get("nsfw") != "True" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Write([]byte(`{"count": 21, "next": "x", "results": [
			{"namespace": "ebkr", "name": "r2modman"},
			{"namespace": "notnotnotswipez", "name": "MoreCompany", "download_count": 100,
			 "last_updated": "2026-04-25T01:28:42.076106Z", "categories": [{"id": "658", "name": "Mods", "slug": "mods"}]}
		]}`))
	})
	mux.HandleFunc("/api/cyberstorm/listing/lethal-company/notnotnotswipez/MoreCompany/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"namespace": "notnotnotswipez", "name": "MoreCompany", "latest_version_number": "1.14.0",
			"dependencies": [{"namespace": "BepInEx", "name": "BepInExPack", "version_number": "5.4.2100"}]}`))
	})
	mux.HandleFunc("/exclusions.md", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("\nebkr-r2modman\n\nOther-Tool\n"))
	})
	mux.HandleFunc("/api/experimental/package/a/b/1.0.0/readme/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"markdown": "# Hi"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &schemaHits
}

func TestSchemaAndFindCommunity(t *testing.T) {
	srv, hits := newTestServer(t)
	cacheDir := t.TempDir()
	ctx := context.Background()

	schema, err := NewClient(srv.URL, "test", cacheDir).Schema(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := schema.FindCommunity("1966720", ""); !ok || c.ID != "lethal-company" || c.Name != "Lethal Company" {
		t.Errorf("by app id: %+v %v", c, ok)
	}
	if c, ok := schema.FindCommunity("", "lethal company.exe"); !ok || c.ID != "lethal-company" {
		t.Errorf("by exe: %+v %v", c, ok)
	}
	if _, ok := schema.FindCommunity("42", "Melon.exe"); ok {
		t.Error("non-BepInEx community must not match")
	}

	// A new client reuses the on-disk cache.
	if _, err := NewClient(srv.URL, "test", cacheDir).Schema(ctx); err != nil || *hits != 1 {
		t.Errorf("cache not used: hits=%d err=%v", *hits, err)
	}

	// Stale cache is used when the server is unreachable.
	old := time.Now().Add(-2 * schemaTTL)
	os.Chtimes(filepath.Join(cacheDir, "ecosystem-schema.json"), old, old)
	if _, err := NewClient("http://127.0.0.1:1", "test", cacheDir).Schema(ctx); err != nil {
		t.Errorf("stale fallback: %v", err)
	}
}

func TestListingAndPackage(t *testing.T) {
	srv, _ := newTestServer(t)
	c := NewClient(srv.URL, "test", t.TempDir())
	c.exclusionsURL = srv.URL + "/exclusions.md"
	ctx := context.Background()

	list, err := c.ListPackages(ctx, "lethal-company", ListOptions{Query: "more", IncludeNSFW: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Packages) != 1 || list.Count != 21 || !list.HasMore || list.Page != 1 || list.Packages[0].Name != "MoreCompany" || list.Packages[0].LastUpdated.IsZero() {
		t.Fatalf("list: %+v", list)
	}

	d, err := c.Package(ctx, "lethal-company", "notnotnotswipez", "MoreCompany")
	if err != nil {
		t.Fatal(err)
	}
	if d.LatestVersion != "1.14.0" || len(d.Dependencies) != 1 || d.PageURL != srv.URL+"/c/lethal-company/p/notnotnotswipez/MoreCompany/" {
		t.Fatalf("detail: %+v", d)
	}

	if _, err := c.Package(ctx, "lethal-company", "no", "such"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing package: %v", err)
	}
	if md, err := c.Readme(ctx, "a", "b", "1.0.0"); err != nil || md != "# Hi" {
		t.Errorf("readme: %q %v", md, err)
	}
}

func TestParseDependency(t *testing.T) {
	ref, err := ParseDependency("some-team-Mod_Name-1.2.30")
	if err != nil || ref != (PackageRef{Namespace: "some-team", Name: "Mod_Name", Version: "1.2.30"}) {
		t.Errorf("got %+v %v", ref, err)
	}
	for _, bad := range []string{"Mod-1.0.0", "a-b-1.0", "a-b/../c-1.0.0", "a-b-1.0.0-beta"} {
		if _, err := ParseDependency(bad); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

func TestExclusionsFallBackToBundledList(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "test", t.TempDir())
	c.exclusionsURL = "http://127.0.0.1:1/exclusions.md"
	ex := c.Exclusions(context.Background())
	if !ex["ebkr-r2modman"] || len(ex) < 10 {
		t.Errorf("bundled exclusions: %v", ex)
	}
}

func TestProfileCodeRoundTrip(t *testing.T) {
	const code = "0d2f8a5c-1b2e-4c3d-9e8f-123456789abc"
	var stored string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/experimental/legacyprofile/create/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		stored = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"key": "` + code + `"}`))
	})
	mux.HandleFunc("GET /api/experimental/legacyprofile/get/"+code+"/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(stored))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := NewClient(srv.URL, "test", t.TempDir())
	ctx := context.Background()

	got, err := c.UploadProfile(ctx, []byte("zip bytes"))
	if err != nil || got != code || !strings.HasPrefix(stored, "#r2modman\n") {
		t.Fatalf("upload: %q %v stored=%q", got, err, stored)
	}
	data, err := c.DownloadProfile(ctx, " "+code+"\n")
	if err != nil || string(data) != "zip bytes" {
		t.Errorf("download: %q %v", data, err)
	}
	if _, err := c.DownloadProfile(ctx, "not-a-code"); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("invalid code: %v", err)
	}
	if _, err := c.DownloadProfile(ctx, "11111111-1111-1111-1111-111111111111"); err == nil {
		t.Error("unknown code must fail")
	}
}
