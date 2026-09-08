package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const DefaultRepo = "JustinNguyen9979/ainovel-cli"
const DefaultCheckInterval = 24 * time.Hour

type CheckOptions struct {
	Repo           string
	CurrentVersion string
	Client         *http.Client
	CachePath      string
	MaxAge         time.Duration
}

type CheckResult struct {
	Latest          string
	Current         string
	Notes           string
	UpdateAvailable bool
	FromCache       bool
}

type checkCache struct {
	LastCheck time.Time `json:"last_check"`
	Latest    string    `json:"latest"`
	Notes     string    `json:"notes"`
}

func CheckUpdate(ctx context.Context, opts CheckOptions) (*CheckResult, error) {
	current := Normalize(opts.CurrentVersion)
	if current == "dev" {
		return &CheckResult{Current: current}, nil
	}
	repo := strings.TrimSpace(opts.Repo)
	if repo == "" {
		repo = DefaultRepo
	}
	maxAge := opts.MaxAge
	if maxAge <= 0 {
		maxAge = DefaultCheckInterval
	}

	var cacheErr error
	if opts.CachePath != "" {
		c, err := loadCache(opts.CachePath)
		switch {
		case err == nil:
			age := time.Since(c.LastCheck)
			if age < 0 {
				cacheErr = fmt.Errorf("thời điểm cache kiểm tra cập nhật nằm trong tương lai: %s", c.LastCheck.Format(time.RFC3339))
			} else if age <= maxAge {
				result, resultErr := c.result(current)
				if resultErr == nil {
					return result, nil
				}
				cacheErr = fmt.Errorf("xác thực cache kiểm tra cập nhật: %w", resultErr)
			}
		case errors.Is(err, os.ErrNotExist):
		default:
			cacheErr = fmt.Errorf("đọc cache kiểm tra cập nhật: %w", err)
		}
	}

	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	rel, err := fetchRelease(ctx, client, repo, "latest")
	if err != nil {
		return nil, errors.Join(cacheErr, err)
	}
	if rel.TagName == "" {
		return nil, errors.Join(cacheErr, fmt.Errorf("release không có tag_name"))
	}
	result, err := newCheckResult(rel.TagName, current, rel.Body, false)
	if err != nil {
		return nil, errors.Join(cacheErr, err)
	}
	if err := writeCache(opts.CachePath, rel); err != nil {
		cacheErr = errors.Join(cacheErr, fmt.Errorf("ghi cache kiểm tra cập nhật: %w", err))
	}
	return result, cacheErr
}

func newCheckResult(latest, current, notes string, fromCache bool) (*CheckResult, error) {
	updateAvailable, err := isNewer(latest, current)
	if err != nil {
		return nil, err
	}
	return &CheckResult{Latest: latest, Current: current, Notes: notes, UpdateAvailable: updateAvailable, FromCache: fromCache}, nil
}

func loadCache(path string) (*checkCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c checkCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("giải mã cache: %w", err)
	}
	if c.Latest == "" || c.LastCheck.IsZero() {
		return nil, fmt.Errorf("cache thiếu latest hoặc last_check")
	}
	return &c, nil
}

func (c *checkCache) result(current string) (*CheckResult, error) {
	return newCheckResult(c.Latest, current, c.Notes, true)
}

func writeCache(path string, rel *release) error {
	if path == "" {
		return nil
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.Marshal(checkCache{LastCheck: time.Now(), Latest: rel.TagName, Notes: rel.Body})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".update-check-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}

func isNewer(latest, current string) (bool, error) {
	latest = Normalize(latest)
	current = Normalize(current)
	if !semver.IsValid(latest) {
		return false, fmt.Errorf("latest version không hợp lệ %q", latest)
	}
	if !semver.IsValid(current) {
		return false, fmt.Errorf("current version không hợp lệ %q", current)
	}
	return semver.Compare(latest, current) > 0, nil
}
