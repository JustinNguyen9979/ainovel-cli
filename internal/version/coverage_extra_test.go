package version

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckUpdateCacheAndFallbackBranches(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	fresh := fmt.Sprintf(`{"latest":"v2.0.0","last_check":%q,"notes":"fresh"}`, time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(cachePath, []byte(fresh), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := CheckUpdate(context.Background(), CheckOptions{Repo: "invalid", CurrentVersion: "v1.0.0", CachePath: cachePath, MaxAge: 24 * time.Hour})
	if err != nil || !result.FromCache || !result.UpdateAvailable {
		t.Fatalf("fresh cache result = %+v/%v", result, err)
	}
	if result.Current != "v1.0.0" || result.Latest != "v2.0.0" {
		t.Fatalf("cache versions = %+v", result)
	}
	if result, err := CheckUpdate(context.Background(), CheckOptions{CurrentVersion: "dev", CachePath: filepath.Join(dir, "missing")}); err != nil || result.Current != "dev" {
		t.Fatalf("dev update check = %+v/%v", result, err)
	}
	future := `{"latest":"v2.0.0","last_check":"2999-01-01T00:00:00Z"}`
	if err := os.WriteFile(cachePath, []byte(future), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckUpdate(context.Background(), CheckOptions{Repo: "invalid", CurrentVersion: "v1.0.0", CachePath: cachePath, Client: &http.Client{Timeout: time.Millisecond}}); err == nil {
		t.Fatal("future cache plus failed network should return error")
	}
	if err := os.WriteFile(cachePath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckUpdate(context.Background(), CheckOptions{Repo: "invalid", CurrentVersion: "v1.0.0", CachePath: cachePath, Client: &http.Client{Timeout: time.Millisecond}}); err == nil {
		t.Fatal("invalid cache plus failed network should return error")
	}
}

func TestVersionCacheAndReleaseHelpers(t *testing.T) {
	if got, err := newCheckResult("v1.2.3", "v1.0.0", "notes", true); err != nil || !got.UpdateAvailable || !got.FromCache {
		t.Fatalf("check result = %+v/%v", got, err)
	}
	if _, err := newCheckResult("bad", "v1.0.0", "", false); err == nil {
		t.Fatal("invalid latest should fail")
	}
	if _, err := newCheckResult("v1.0.0", "bad", "", false); err == nil {
		t.Fatal("invalid current should fail")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	if err := os.WriteFile(path, []byte(`{"latest":"v2.0.0","last_check":"2026-09-06T00:00:00Z","notes":"notes"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cache, err := loadCache(path)
	if err != nil || cache.Latest != "v2.0.0" {
		t.Fatalf("load cache = %+v/%v", cache, err)
	}
	for _, raw := range []string{"{", `{"latest":""}`, `{"latest":"v1"}`} {
		badPath := filepath.Join(dir, fmt.Sprintf("bad-%d", len(raw)))
		if err := os.WriteFile(badPath, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadCache(badPath); err == nil {
			t.Fatalf("invalid cache accepted: %s", raw)
		}
	}
	if err := writeCache(filepath.Join(dir, "nested", "cache.json"), &release{TagName: "v2.0.0", Body: "notes"}); err != nil {
		t.Fatal(err)
	}
	if err := writeCache("", &release{}); err != nil {
		t.Fatal(err)
	}
	if err := writeCache(filepath.Join(dir, "cache2.json"), &release{TagName: "v2.0.0"}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		latest, current string
		want            bool
	}{
		{"v2.0.0", "v1.0.0", true}, {"v1.0.0", "v2.0.0", false}, {"v1.0.0", "v1.0.0", false},
	} {
		got, err := isNewer(tc.latest, tc.current)
		if err != nil || got != tc.want {
			t.Errorf("isNewer(%q,%q) = %v/%v", tc.latest, tc.current, got, err)
		}
	}
}

type rewriteTransport struct {
	target string
	base   http.RoundTripper
}

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, err := url.Parse(r.target)
	if err != nil {
		return nil, err
	}
	copyReq := req.Clone(req.Context())
	copyReq.URL.Scheme = target.Scheme
	copyReq.URL.Host = target.Host
	return r.base.RoundTrip(copyReq)
}

func TestVersionHTTPAndArchiveHelpers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "bad") {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		if strings.Contains(r.URL.Path, "invalid") {
			_, _ = w.Write([]byte("{"))
			return
		}
		_, _ = w.Write([]byte(`{"tag_name":"v2.0.0","body":"notes"}`))
	}))
	defer server.Close()
	client := server.Client()
	serverURL := server.URL
	client.Transport = rewriteTransport{target: serverURL, base: client.Transport}
	if rel, err := fetchRelease(context.Background(), client, "ok", "latest"); err != nil || rel.TagName != "v2.0.0" {
		t.Fatalf("fetch release = %+v/%v", rel, err)
	}
	if _, err := fetchRelease(context.Background(), client, "bad", "latest"); err == nil {
		t.Fatal("bad status should fail")
	}
	if _, err := fetchRelease(context.Background(), client, "invalid", "latest"); err == nil {
		t.Fatal("invalid response should fail")
	}
	targetClient := server.Client()
	targetClient.Transport = rewriteTransport{target: serverURL, base: targetClient.Transport}
	dst := filepath.Join(t.TempDir(), "download")
	if err := download(context.Background(), client, server.URL+"/ok", dst, 0); err != nil {
		t.Fatal(err)
	}
	if err := download(context.Background(), client, server.URL+"/bad", dst, 0); err == nil {
		t.Fatal("download bad status should fail")
	}
	archive := filepath.Join(t.TempDir(), "pkg.tar.gz")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tarWriter := tar.NewWriter(gz)
	if err := tarWriter.WriteHeader(&tar.Header{Name: "other", Mode: 0o755, Size: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.WriteHeader(&tar.Header{Name: "ainovel-cli", Mode: 0o755, Size: 3}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write([]byte("new")); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := extractBinary(archive, filepath.Dir(archive), "ainovel-cli")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	if string(data) != "new" {
		t.Fatalf("extracted = %q", data)
	}
	if _, err := extractBinary(archive, filepath.Dir(archive), "missing"); err == nil {
		t.Fatal("missing binary should fail")
	}
	sum := sha256.Sum256([]byte("new"))
	checksum := filepath.Join(filepath.Dir(archive), "checksums.txt")
	if err := os.WriteFile(checksum, []byte(fmt.Sprintf("%x  ainovel-cli\n", sum)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(out, checksum, "ainovel-cli"); err != nil {
		t.Fatal(err)
	}
}
