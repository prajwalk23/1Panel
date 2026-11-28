package service

// fork: the "new panel version" alert follows the fork's own GitHub Releases on fork builds,
// like the footer Update button. Plain upstream builds keep the official update check.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/common"
	"github.com/1Panel-dev/1Panel/agent/utils/req_helper"
	versionUtil "github.com/1Panel-dev/1Panel/agent/utils/version"
)

var (
	forkVersionPattern = regexp.MustCompile(`^v\d{1,4}\.\d{1,4}\.\d{1,4}\.\d{1,6}$`)
	forkRepoPattern    = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9_.-]*/[A-Za-z0-9_-][A-Za-z0-9_.-]*$`)
)

// loadPanelUpdateVersion returns the version to alert about, or "" when the panel is up to date.
func loadPanelUpdateVersion() (string, error) {
	// Core publishes the full fork version here; it is a plain base version on upstream builds.
	running, _ := getSettingValue("PanelRunningVersion")
	if !forkVersionPattern.MatchString(running) {
		info, err := versionUtil.GetUpgradeVersionInfo()
		if err != nil {
			return "", err
		}
		return getValidVersion(info), nil
	}
	if global.CONF.Base.IsOffline {
		return "", nil
	}
	repo, err := getSettingValue("PanelForkRepo")
	if err != nil {
		return "", err
	}
	if !forkRepoPattern.MatchString(repo) {
		return "", fmt.Errorf("invalid fork repository name %q", repo)
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	_, body, err := req_helper.HandleRequest(apiURL, http.MethodGet, constant.TimeOut20s)
	if err != nil {
		return "", fmt.Errorf("load latest release of %s failed: %w", repo, err)
	}
	return newerForkRelease(running, body)
}

// newerForkRelease reads a GitHub release and returns its tag when it is newer than running.
func newerForkRelease(running string, body []byte) (string, error) {
	var release struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("decode fork release: %w", err)
	}
	if release.Draft || release.Prerelease || !forkVersionPattern.MatchString(release.TagName) {
		return "", nil
	}
	if !common.ComparePanelVersion(release.TagName, running) {
		return "", nil
	}
	return release.TagName, nil
}
