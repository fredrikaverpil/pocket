package shim

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GoChecksums holds SHA256 checksums for Go downloads, keyed by "os-arch".
type GoChecksums map[string]string

// goRelease represents a Go release from the download API.
type goRelease struct {
	Version string   `json:"version"`
	Files   []goFile `json:"files"`
}

// goFile represents a downloadable file in a Go release.
type goFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	SHA256   string `json:"sha256"`
	Kind     string `json:"kind"`
}

// fetchGoChecksums fetches SHA256 checksums for the given Go version
// from the official Go download API. Works with stable releases, RCs, and betas.
func fetchGoChecksums(ctx context.Context, version string) (GoChecksums, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://go.dev/dl/?mode=json&include=all", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Go downloads: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var releases []goRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Find the matching version (API uses "go" prefix).
	targetVersion := "go" + version
	for _, release := range releases {
		if release.Version != targetVersion {
			continue
		}

		checksums := make(GoChecksums)
		for _, file := range release.Files {
			// Only include archives (tar.gz for unix, zip for windows).
			if file.Kind != "archive" {
				continue
			}
			if file.SHA256 == "" {
				continue
			}
			key := file.OS + "-" + file.Arch
			checksums[key] = file.SHA256
		}
		return checksums, nil
	}

	return nil, fmt.Errorf("version %s not found", version)
}
