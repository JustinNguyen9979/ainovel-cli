package version

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type checkRoundTrip func(*http.Request) (*http.Response, error)

func (f checkRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func checkHTTPResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func TestCheckUpdateVersionComparison(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
		wantErr         bool
	}{
		{"v1.2.4", "v1.2.3", true, false},
		{"v1.2.3", "v1.2.3", false, false},
		{"v1.2.3-rc.1", "v1.2.2", true, false},
		{"nightly", "v1.0.0", false, true},
	}
	for _, tc := range cases {
		got, err := isNewer(tc.latest, tc.current)
		if got != tc.want || (err != nil) != tc.wantErr {
			t.Fatalf("isNewer(%q, %q) = %v, %v", tc.latest, tc.current, got, err)
		}
	}
}

func TestCheckUpdateFetchesAndCaches(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: checkRoundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		return checkHTTPResponse(http.StatusOK, `{"tag_name":"v1.2.4","body":"## New Features\n- update"}`), nil
	})}
	cachePath := filepath.Join(t.TempDir(), "update-check.json")
	opts := CheckOptions{CurrentVersion: "v1.2.3", Client: client, CachePath: cachePath}
	result, err := CheckUpdate(context.Background(), opts)
	if err != nil || result == nil || !result.UpdateAvailable || result.FromCache {
		t.Fatalf("first check = %+v, %v", result, err)
	}
	result, err = CheckUpdate(context.Background(), opts)
	if err != nil || result == nil || !result.FromCache || calls != 1 {
		t.Fatalf("cached check = %+v, %v, calls=%d", result, err, calls)
	}
}

func TestCheckUpdateDevAndInvalidCache(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: checkRoundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		return checkHTTPResponse(http.StatusOK, `{"tag_name":"v9.9.9"}`), nil
	})}
	result, err := CheckUpdate(context.Background(), CheckOptions{CurrentVersion: "dev", Client: client})
	if err != nil || result == nil || calls != 0 {
		t.Fatalf("dev check = %+v, %v, calls=%d", result, err, calls)
	}
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	if err := os.WriteFile(cachePath, []byte(`{broken`), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err = CheckUpdate(context.Background(), CheckOptions{CurrentVersion: "v1.2.3", Client: client, CachePath: cachePath})
	if result == nil || err == nil || calls != 1 {
		t.Fatalf("corrupt cache recovery = %+v, %v, calls=%d", result, err, calls)
	}
	var cache checkCache
	data, readErr := os.ReadFile(cachePath)
	if readErr != nil || json.Unmarshal(data, &cache) != nil || cache.Latest != "v9.9.9" {
		t.Fatalf("repaired cache invalid: %s, %v", data, readErr)
	}
}

func TestCheckUpdateCacheExpiry(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: checkRoundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		return checkHTTPResponse(http.StatusOK, `{"tag_name":"v1.2.4"}`), nil
	})}
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	data, _ := json.Marshal(checkCache{LastCheck: time.Now().Add(-2 * time.Hour), Latest: "v1.2.3"})
	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := CheckUpdate(context.Background(), CheckOptions{CurrentVersion: "v1.2.2", Client: client, CachePath: cachePath, MaxAge: time.Hour})
	if err != nil || result == nil || result.FromCache || calls != 1 {
		t.Fatalf("expired cache = %+v, %v, calls=%d", result, err, calls)
	}
}
