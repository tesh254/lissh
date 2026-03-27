package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/blang/semver/v4"
)

const devVersion = "dev"

var (
	Version   = devVersion
	Commit    = ""
	BuildDate = ""
)

func Get() semver.Version {
	if Version == devVersion {
		return semver.MustParse("0.0.0")
	}
	v, err := semver.Parse(strings.TrimPrefix(Version, "v"))
	if err != nil {
		return semver.MustParse("0.0.0")
	}
	return v
}

func String() string {
	if Version == devVersion {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			return devVersion
		}
		commit := ""
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				commit = s.Value
				break
			}
		}
		if commit == "" {
			commit = "unknown"
		}
		return fmt.Sprintf("%s (commit %s)", devVersion, commit[:7])
	}
	return Version
}

type ReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func CheckForUpdate() (semver.Version, error) {
	const latestURL = "https://api.github.com/repos/tesh254/lissh/releases/latest"

	resp, err := http.Get(latestURL)
	if err != nil {
		return semver.MustParse("0.0.0"), fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return semver.MustParse("0.0.0"), fmt.Errorf("failed to check for updates: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return semver.MustParse("0.0.0"), fmt.Errorf("failed to read response: %w", err)
	}

	var release ReleaseResponse
	if err := json.Unmarshal(body, &release); err != nil {
		return semver.MustParse("0.0.0"), fmt.Errorf("failed to parse response: %w", err)
	}

	tag := strings.TrimPrefix(release.TagName, "v")
	latest, err := semver.Parse(tag)
	if err != nil {
		return semver.MustParse("0.0.0"), fmt.Errorf("failed to parse version: %w", err)
	}

	return latest, nil
}
