// Package selfupdate checks GitHub releases for a newer agentboard build
// and, if one exists, downloads and atomically replaces the running
// executable.
package selfupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const DefaultRepo = "ayan-de/agent-board"

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Fetcher abstracts the GitHub API and asset download so Check/Apply are
// testable without network access.
type Fetcher interface {
	FetchLatestRelease() (*Release, error)
	Download(url string) (io.ReadCloser, error)
}

type httpFetcher struct {
	repo   string
	client *http.Client
}

// NewHTTPFetcher returns a Fetcher backed by the real GitHub API, e.g.
// NewHTTPFetcher(DefaultRepo).
func NewHTTPFetcher(repo string) Fetcher {
	return &httpFetcher{
		repo:   repo,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *httpFetcher) FetchLatestRelease() (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", f.repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("selfupdate.FetchLatestRelease: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("selfupdate.FetchLatestRelease: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("selfupdate.FetchLatestRelease: unexpected status %s", resp.Status)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("selfupdate.FetchLatestRelease: decoding response: %w", err)
	}
	return &rel, nil
}

func (f *httpFetcher) Download(url string) (io.ReadCloser, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("selfupdate.Download: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("selfupdate.Download: unexpected status %s", resp.Status)
	}
	return resp.Body, nil
}

// AssetName returns the release asset filename for the given GOOS/GOARCH,
// matching the naming produced by .github/workflows/release.yml.
func AssetName(goos, goarch string) (string, error) {
	if goos == "windows" {
		return "", fmt.Errorf("selfupdate: windows is not supported yet")
	}

	var arch string
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "arm64"
	default:
		return "", fmt.Errorf("selfupdate: unsupported architecture %q", goarch)
	}

	return fmt.Sprintf("agentboard-%s-%s", goos, arch), nil
}

// Check fetches the latest release and reports whether it differs from
// currentVersion. A currentVersion of "dev" (unset/local build) always
// needs an update.
func Check(f Fetcher, currentVersion string) (release *Release, needsUpdate bool, err error) {
	rel, err := f.FetchLatestRelease()
	if err != nil {
		return nil, false, fmt.Errorf("selfupdate.Check: %w", err)
	}
	needsUpdate = currentVersion == "dev" || currentVersion != rel.TagName
	return rel, needsUpdate, nil
}

// Apply downloads the release asset matching goos/goarch and atomically
// replaces the executable at execPath. The existing executable is left
// untouched if any step fails.
func Apply(f Fetcher, release *Release, execPath, goos, goarch string) error {
	assetName, err := AssetName(goos, goarch)
	if err != nil {
		return fmt.Errorf("selfupdate.Apply: %w", err)
	}

	var asset *Asset
	for i := range release.Assets {
		if release.Assets[i].Name == assetName {
			asset = &release.Assets[i]
			break
		}
	}
	if asset == nil {
		return fmt.Errorf("selfupdate.Apply: no release asset named %q", assetName)
	}

	body, err := f.Download(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("selfupdate.Apply: %w", err)
	}
	defer body.Close()

	dir := filepath.Dir(execPath)
	tmp, err := os.CreateTemp(dir, ".agentboard-update-*")
	if err != nil {
		return fmt.Errorf("selfupdate.Apply: creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once renamed into place

	if _, err := io.Copy(tmp, body); err != nil {
		tmp.Close()
		return fmt.Errorf("selfupdate.Apply: downloading %s: %w", assetName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("selfupdate.Apply: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("selfupdate.Apply: %w", err)
	}
	if err := os.Rename(tmpPath, execPath); err != nil {
		return fmt.Errorf("selfupdate.Apply: replacing executable: %w", err)
	}

	return nil
}
