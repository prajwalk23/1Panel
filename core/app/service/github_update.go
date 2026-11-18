package service

// fork: updates the panel from the fork's own GitHub Releases instead of the
// official update servers. The install steps mirror UpgradeService.Upgrade()
// (see FORK_PATCHES.md) and reuse its backup and rollback helpers.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/1Panel-dev/1Panel/core/app/model"
	"github.com/1Panel-dev/1Panel/core/buserr"
	"github.com/1Panel-dev/1Panel/core/constant"
	"github.com/1Panel-dev/1Panel/core/global"
	"github.com/1Panel-dev/1Panel/core/utils/controller"
	"github.com/1Panel-dev/1Panel/core/utils/ctl_conf"
	"github.com/1Panel-dev/1Panel/core/utils/files"
	githubver "github.com/1Panel-dev/1Panel/core/utils/github"
	"github.com/1Panel-dev/1Panel/core/utils/req_helper"
)

const forkChecksumFile = "checksums.txt"

type GithubUpdateService struct{}

type IGithubUpdateService interface {
	Info() (*dto.ForkUpdateInfo, error)
	Upgrade(req dto.ForkUpgrade) error
}

func NewIGithubUpdateService() IGithubUpdateService {
	return &GithubUpdateService{}
}

type forkReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type forkRelease struct {
	TagName     string             `json:"tag_name"`
	Name        string             `json:"name"`
	Body        string             `json:"body"`
	HTMLURL     string             `json:"html_url"`
	PublishedAt string             `json:"published_at"`
	Draft       bool               `json:"draft"`
	Prerelease  bool               `json:"prerelease"`
	Assets      []forkReleaseAsset `json:"assets"`
}

func (r *forkRelease) asset(name string) *forkReleaseAsset {
	for i := range r.Assets {
		if r.Assets[i].Name == name {
			return &r.Assets[i]
		}
	}
	return nil
}

func loadForkRelease(tag string) (*forkRelease, error) {
	if !githubver.ValidRepo(global.ForkRepo) {
		return nil, buserr.New("ErrForkRepoInvalid")
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", global.ForkRepo)
	if tag != "" {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", global.ForkRepo, url.PathEscape(tag))
	}
	code, body, err := req_helper.HandleRequestWithProxy(apiURL, http.MethodGet, constant.TimeOut20s)
	if err != nil {
		return nil, buserr.WithName("ErrForkGithubAPI", err.Error())
	}
	if code == http.StatusNotFound {
		return nil, buserr.New("ErrForkReleaseNotFound")
	}
	if code != http.StatusOK {
		return nil, buserr.WithName("ErrForkGithubAPI", fmt.Sprintf("HTTP %d", code))
	}
	var release forkRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, buserr.WithName("ErrForkGithubAPI", err.Error())
	}
	return &release, nil
}

func (g *GithubUpdateService) Info() (*dto.ForkUpdateInfo, error) {
	panelVersion, _ := settingRepo.GetValueByKey("SystemVersion")
	status, _ := settingRepo.GetValueByKey("SystemStatus")
	info := &dto.ForkUpdateInfo{
		IsForkBuild:  global.IsForkBuild(),
		PanelVersion: panelVersion,
		ForkVersion:  global.ForkVersion,
		ForkRepo:     global.ForkRepo,
		SystemStatus: status,
	}
	if arch, err := loadArch(); err == nil {
		info.Arch = arch
	}
	if global.CONF.Base.IsOffline {
		return info, nil
	}
	release, err := loadForkRelease("")
	if err != nil {
		return nil, err
	}
	if _, ok := githubver.Parse(release.TagName); !ok || release.Draft || release.Prerelease {
		return info, nil
	}
	info.LatestVersion = release.TagName
	info.LatestBase = githubver.Base(release.TagName)
	info.ReleaseName = release.Name
	info.ReleaseNote = release.Body
	info.ReleaseURL = release.HTMLURL
	info.PublishedAt = release.PublishedAt
	info.HasUpdate = global.IsForkBuild() && githubver.Newer(release.TagName, global.ForkVersion)
	info.BaseChange = info.HasUpdate && info.LatestBase != githubver.Base(global.ForkVersion)
	if info.Arch != "" {
		info.AssetReady = release.asset(githubver.AssetName(release.TagName, info.Arch)) != nil && release.asset(forkChecksumFile) != nil
	}
	return info, nil
}

func (g *GithubUpdateService) Upgrade(req dto.ForkUpgrade) error {
	if !global.IsForkBuild() {
		return buserr.New("ErrForkNotForkBuild")
	}
	version := strings.TrimSpace(req.Version)
	if _, ok := githubver.Parse(version); !ok {
		return buserr.New("ErrForkInvalidVersion")
	}
	if !githubver.Newer(version, global.ForkVersion) {
		return buserr.New("ErrForkNotNewer")
	}
	if status, _ := settingRepo.GetValueByKey("SystemStatus"); status == "Upgrading" {
		return buserr.New("ErrForkBusy")
	}
	arch, err := loadArch()
	if err != nil {
		return err
	}
	svcInfo, err := loadServiceInfo()
	if err != nil {
		return err
	}
	if err := checkUpgradeSpace(); err != nil {
		return err
	}
	release, err := loadForkRelease(version)
	if err != nil {
		return err
	}
	fileName := githubver.AssetName(version, arch)
	pkg := release.asset(fileName)
	if pkg == nil {
		return buserr.WithName("ErrForkAssetMissing", fileName)
	}
	sums := release.asset(forkChecksumFile)
	if sums == nil {
		return buserr.WithName("ErrForkAssetMissing", forkChecksumFile)
	}

	baseDir := path.Join(global.CONF.Base.InstallDir, "1panel/tmp/upgrade", version)
	downloadDir := path.Join(baseDir, "downloads")
	originalDir := path.Join(baseDir, "original")
	_ = os.RemoveAll(baseDir)
	if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(originalDir, os.ModePerm); err != nil {
		return err
	}

	global.LOG.Infof("[fork] start to upgrade %s -> %s", global.ForkVersion, version)
	_ = settingRepo.Update("SystemStatus", "Upgrading")
	go installForkPackage(version, pkg.BrowserDownloadURL, sums.BrowserDownloadURL, fileName, baseDir, svcInfo)
	return nil
}

func installForkPackage(version, packageURL, sumsURL, fileName, baseDir string, svcInfo serviceInfo) {
	u := &UpgradeService{}
	downloadDir := path.Join(baseDir, "downloads")
	originalDir := path.Join(baseDir, "original")
	fail := func(format string, args ...interface{}) {
		global.LOG.Errorf("[fork] "+format, args...)
		_ = settingRepo.Update("SystemStatus", "Free")
	}
	oldLang := ctl_conf.Load("LANGUAGE")

	sumsPath := path.Join(downloadDir, forkChecksumFile)
	if err := files.DownloadFileWithProxyStream(sumsURL, sumsPath); err != nil {
		fail("download %s failed, err: %v", forkChecksumFile, err)
		return
	}
	pkgPath := path.Join(downloadDir, fileName)
	if err := files.DownloadFileWithProxyStream(packageURL, pkgPath); err != nil {
		fail("download %s failed, err: %v", fileName, err)
		return
	}
	sumsFile, err := os.Open(sumsPath)
	if err != nil {
		fail("open %s failed, err: %v", forkChecksumFile, err)
		return
	}
	want := githubver.ParseChecksums(sumsFile)[fileName]
	_ = sumsFile.Close()
	got, err := githubver.FileSHA256(pkgPath)
	if err != nil || want == "" || got != want {
		fail("checksum mismatch for %s (want %q, got %q, err %v)", fileName, want, got, err)
		_ = os.RemoveAll(downloadDir)
		return
	}
	global.LOG.Infof("[fork] %s verified (sha256 %s)", fileName, got)

	if err := files.HandleUnTar(pkgPath, downloadDir, ""); err != nil {
		fail("decompress file failed, err: %v", err)
		return
	}
	tmpDir := path.Join(downloadDir, strings.TrimSuffix(fileName, ".tar.gz"))
	for _, item := range []string{"1panel-core", "1panel-agent", "1pctl", "initscript", "lang", "GeoIP.mmdb"} {
		if _, err := os.Stat(path.Join(tmpDir, item)); err != nil {
			fail("package is missing %s", item)
			return
		}
	}

	if err := u.handleBackup(originalDir, svcInfo); err != nil {
		fail("handle backup original file failed, err: %v", err)
		return
	}
	itemLog := model.UpgradeLog{NodeID: 0, OldVersion: global.ForkVersion, NewVersion: version, BackupFile: baseDir}
	_ = upgradeLogRepo.Create(&itemLog)
	global.LOG.Info("[fork] backup original data successful, now start to upgrade!")

	if err := files.CopyFileWithRename(path.Join(tmpDir, "1panel-core"), "/usr/local/bin/1panel-core"); err != nil {
		fail("upgrade 1panel-core failed, err: %v", err)
		u.handleRollback(originalDir, 1, svcInfo)
		return
	}
	if err := files.CopyFileWithRename(path.Join(tmpDir, "1panel-agent"), "/usr/local/bin/1panel-agent"); err != nil {
		fail("upgrade 1panel-agent failed, err: %v", err)
		u.handleRollback(originalDir, 1, svcInfo)
		return
	}
	if err := files.CopyFileWithRename(path.Join(tmpDir, "1pctl"), "/usr/local/bin/1pctl"); err != nil {
		fail("upgrade 1pctl failed, err: %v", err)
		u.handleRollback(originalDir, 2, svcInfo)
		return
	}
	if err := ctl_conf.UpdateInFile("/usr/local/bin/1pctl", "BASE_DIR", global.CONF.Base.InstallDir); err != nil {
		fail("upgrade basedir in 1pctl failed, err: %v", err)
		u.handleRollback(originalDir, 2, svcInfo)
		return
	}
	if err := ctl_conf.UpdateInFile("/usr/local/bin/1pctl", "LANGUAGE", oldLang); err != nil {
		fail("upgrade language in 1pctl failed, err: %v", err)
		u.handleRollback(originalDir, 2, svcInfo)
		return
	}
	initScriptPath := path.Join(tmpDir, "initscript")
	if err := files.CopyItem(false, true, path.Join(initScriptPath, svcInfo.selCoreName), svcInfo.basePath); err != nil {
		fail("upgrade %s failed, err: %v", svcInfo.coreName, err)
		u.handleRollback(originalDir, 3, svcInfo)
		return
	}
	if err := files.CopyItem(false, true, path.Join(initScriptPath, svcInfo.selAgentName), svcInfo.basePath); err != nil {
		fail("upgrade %s failed, err: %v", svcInfo.agentName, err)
		u.handleRollback(originalDir, 3, svcInfo)
		return
	}
	if err := files.CopyItem(true, true, path.Join(tmpDir, "lang"), "/usr/local/bin"); err != nil {
		fail("update language files failed: %v", err)
		u.handleRollback(originalDir, 4, svcInfo)
		return
	}
	if err := files.CopyFileWithRename(path.Join(tmpDir, "GeoIP.mmdb"), path.Join(global.CONF.Base.InstallDir, "1panel/geo/GeoIP.mmdb")); err != nil {
		fail("update GeoIP database failed: %v", err)
		u.handleRollback(originalDir, 4, svcInfo)
		return
	}

	global.LOG.Infof("[fork] upgrade to %s successful!", version)
	dropBackupCopies()
	base := githubver.Base(version)
	_ = settingRepo.Update("SystemVersion", base)
	_ = global.AgentDB.Model(&model.Setting{}).Where("key = ?", "SystemVersion").Updates(map[string]interface{}{"value": base}).Error
	global.CONF.Base.Version = base
	_ = os.RemoveAll(downloadDir)
	_ = settingRepo.Update("SystemStatus", "Free")

	controller.RestartPanel(true, true, true)
}
