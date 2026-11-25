package service

// fork: GitHub section - projects and settings. The webhook lives in github_hook.go.

import (
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/buserr"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/cmd"
	"github.com/1Panel-dev/1Panel/agent/utils/encrypt"
	githubutil "github.com/1Panel-dev/1Panel/agent/utils/github"
)

var githubRepo = repo.NewIGithubRepo()

const (
	githubSettingShell       = "GithubDefaultShell"
	githubSettingRetention   = "GithubLogRetention"
	githubSettingSharedKeyID = "GithubSharedKeyID"

	githubDefaultTimeout     = 600
	githubDefaultRetention   = 50
	githubDefaultKeepRecords = 5
	githubMaxKeepRecords     = 100
)

var githubUserPattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}\$?$`)

type GithubService struct{}

type IGithubService interface {
	SearchProjects(req dto.GithubProjectSearch) (int64, []dto.GithubProjectInfo, error)
	CreateProject(req dto.GithubProjectOperate) (*dto.GithubProjectInfo, error)
	UpdateProject(req dto.GithubProjectOperate) error
	DeleteProject(id uint) error
	UpdateProjectStatus(req dto.GithubProjectStatus) error
	LoadLinked(req dto.GithubLinkedSearch) (*dto.GithubProjectInfo, error)
	LoadSummary(id uint) (*dto.GithubProjectSummary, error)
	SearchCommits(req dto.GithubCommitSearch) (int64, []dto.GithubCommitInfo, error)
	RefreshWebhookToken(id uint) (*dto.GithubProjectSecret, error)
	LoadWebhookInfo(id uint) (*dto.GithubWebhookInfo, error)
	ConfigureWebhook(req dto.GithubWebhookConfig) (*dto.GithubProjectSecret, error)
	Deploy(req dto.GithubProjectDeploy) error
	Cancel(id uint) error

	ListScripts(req dto.GithubScriptList) ([]dto.GithubScriptInfo, error)
	DefaultScript(req dto.GithubScriptList) (string, error)
	CreateScript(req dto.GithubScriptOperate) (*dto.GithubScriptInfo, error)
	UpdateScript(req dto.GithubScriptOperate) error
	DeleteScript(id uint) error
	ListSystemUsers() ([]string, error)

	SearchLogs(req dto.GithubLogSearch) (int64, []dto.GithubLogInfo, error)
	CleanLogs(req dto.GithubLogClean) error

	SearchKeys(req dto.GithubKeySearch) (int64, []dto.GithubKeyInfo, error)
	CreateKey(req dto.GithubKeyCreate) (*dto.GithubKeyInfo, error)
	ImportKey(req dto.GithubKeyImport) (*dto.GithubKeyInfo, error)
	RegenerateKey(id uint) (*dto.GithubKeyInfo, error)
	DeleteKey(id uint) error
	ListKeyOptions() ([]dto.GithubKeyOption, error)

	LoadSetting() (*dto.GithubSettingInfo, error)
	UpdateSetting(req dto.GithubSetting) error
	CheckRepo(req dto.GithubRepoCheck) (*dto.GithubRepoCheckInfo, error)
}

func NewIGithubService() IGithubService {
	return &GithubService{}
}

// ---------- projects ----------

func (g *GithubService) SearchProjects(req dto.GithubProjectSearch) (int64, []dto.GithubProjectInfo, error) {
	opts := []repo.DBOption{repo.WithByLikeName(req.Info), repo.WithOrderDesc("created_at")}
	if req.Standalone {
		opts = append(opts, githubRepo.WithStandalone())
	}
	total, items, err := githubRepo.PageProjects(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	setting := loadGithubSetting()
	list := make([]dto.GithubProjectInfo, 0, len(items))
	for _, item := range items {
		list = append(list, githubProjectInfo(item, setting))
	}
	return total, list, nil
}

func (g *GithubService) CreateProject(req dto.GithubProjectOperate) (*dto.GithubProjectInfo, error) {
	if existing, _ := githubRepo.GetProject(repo.WithByName(strings.TrimSpace(req.Name))); existing.ID != 0 {
		return nil, buserr.New("ErrGithubNameExist")
	}
	project := model.GithubProject{}
	if err := githubApplyProject(&project, req); err != nil {
		return nil, err
	}
	// The webhook is configured later, from the Git Manager or when editing the project.
	project.DomainMode = dto.GithubDomainNone

	createdKeyID := uint(0)
	if project.KeyMode == dto.GithubKeyModeProject {
		keyID, created, err := githubPickProjectKey(project.Name, req.KeyID, 0)
		if err != nil {
			return nil, err
		}
		project.KeyID = keyID
		if created {
			createdKeyID = keyID
		}
	}
	if err := githubRepo.CreateProject(&project); err != nil {
		githubDropKey(createdKeyID)
		return nil, err
	}
	info := githubProjectInfo(project, loadGithubSetting())
	return &info, nil
}

func (g *GithubService) UpdateProject(req dto.GithubProjectOperate) error {
	project, err := githubRepo.GetProject(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New("ErrRecordNotFound")
	}
	if existing, _ := githubRepo.GetProject(repo.WithByName(strings.TrimSpace(req.Name))); existing.ID != 0 && existing.ID != project.ID {
		return buserr.New("ErrGithubNameExist")
	}
	oldKeyMode, oldKeyID := project.KeyMode, project.KeyID
	if project.WebsiteID != 0 || project.RuntimeID != 0 {
		// A linked project keeps its name and folder, they belong to the website or runtime.
		req.Name, req.ProjectPath = project.Name, project.ProjectPath
	}
	if err := githubApplyProject(&project, req); err != nil {
		return err
	}

	createdKeyID := uint(0)
	if project.KeyMode == dto.GithubKeyModeProject {
		current := uint(0)
		if oldKeyMode == dto.GithubKeyModeProject {
			current = oldKeyID
		}
		keyID, created, err := githubPickProjectKey(project.Name, req.KeyID, current)
		if err != nil {
			return err
		}
		project.KeyID = keyID
		if created {
			createdKeyID = keyID
		}
	}
	if err := githubRepo.SaveProject(&project); err != nil {
		githubDropKey(createdKeyID)
		return err
	}
	if oldKeyMode == dto.GithubKeyModeProject && oldKeyID != 0 && project.KeyID != oldKeyID {
		githubDropKey(oldKeyID)
	}
	return nil
}

func (g *GithubService) DeleteProject(id uint) error {
	project, err := githubRepo.GetProject(repo.WithByID(id))
	if err != nil {
		return buserr.New("ErrRecordNotFound")
	}
	return githubRemoveProject(project)
}

// githubRemoveProject stops a running deploy and removes the project with its scripts, logs and key.
func githubRemoveProject(project model.GithubProject) error {
	id := project.ID
	if ok, dropped := githubQueue.Cancel(id); ok || dropped != nil {
		githubMarkDropped(dropped, "GithubCanceled")
		githubQueue.WaitProject(id, 30*time.Second)
	}
	if err := githubRepo.DeleteProject(repo.WithByID(id)); err != nil {
		return err
	}
	logs, _ := githubRepo.ListLogs(githubRepo.WithByProjectID(id))
	githubDeleteLogs(logs)
	_ = githubRepo.DeleteScript(githubRepo.WithByProjectID(id))
	if project.KeyMode == dto.GithubKeyModeProject {
		githubDropKey(project.KeyID)
	}
	return nil
}

func (g *GithubService) UpdateProjectStatus(req dto.GithubProjectStatus) error {
	if _, err := githubRepo.GetProject(repo.WithByID(req.ID)); err != nil {
		return buserr.New("ErrRecordNotFound")
	}
	return githubRepo.UpdateProject(req.ID, map[string]interface{}{"enabled": req.Enabled})
}

// githubApplyProject validates req and copies it onto project.
func githubApplyProject(project *model.GithubProject, req dto.GithubProjectOperate) error {
	name := strings.TrimSpace(req.Name)
	if name == "" || strings.ContainsAny(name, "/\\\x00") {
		return buserr.New("ErrGithubInvalidName")
	}
	cloneURL := strings.TrimSpace(req.CloneURL)
	if !githubValidCloneURL(cloneURL) {
		return buserr.New("ErrGithubInvalidCloneURL")
	}
	repoName := strings.TrimSpace(req.Repo)
	if repoName == "" {
		repoName = githubutil.RepoFromCloneURL(cloneURL)
	}
	if !githubutil.ValidRepo(repoName) {
		return buserr.New("ErrGithubInvalidRepo")
	}
	branch := strings.TrimSpace(req.Branch)
	if !githubutil.ValidBranch(branch) {
		return buserr.New("ErrGithubInvalidBranch")
	}
	projectPath, err := githubCleanProjectPath(req.ProjectPath)
	if err != nil {
		return err
	}
	switch req.KeyMode {
	case dto.GithubKeyModeShared:
		setting := loadGithubSetting()
		if setting.SharedKeyID == 0 {
			return buserr.New("ErrGithubSharedKeyNotSet")
		}
		project.KeyID = 0
	case dto.GithubKeyModeCustom:
		// "custom" means: use the key chosen by ID (generated or imported).
		key, err := githubRepo.GetKey(repo.WithByID(req.KeyID))
		if err != nil {
			return buserr.New("ErrGithubKeyNotFound")
		}
		project.KeyID = key.ID
	case dto.GithubKeyModeNone:
		if githubutil.IsSSHCloneURL(cloneURL) {
			return buserr.New("ErrGithubKeyRequired")
		}
		project.KeyID = 0
	case dto.GithubKeyModeToken:
		if !githubutil.IsHTTPSCloneURL(cloneURL) {
			return buserr.New("ErrGithubTokenNeedsHTTPS")
		}
		token := strings.TrimSpace(req.Token)
		if token == "" && project.Token == "" {
			return buserr.New("ErrGithubTokenRequired")
		}
		if strings.ContainsAny(token, " \t\r\n\x00") || strings.ContainsAny(req.TokenUser, " \t\r\n\x00:") {
			return buserr.New("ErrGithubInvalidToken")
		}
		if token != "" {
			encrypted, err := encrypt.StringEncrypt(token)
			if err != nil {
				return err
			}
			project.Token = encrypted
		}
		project.TokenUser = strings.TrimSpace(req.TokenUser)
		project.KeyID = 0
	case dto.GithubKeyModeProject:
		// key is created or kept by the caller
	default:
		return buserr.New("ErrGithubInvalidKeyMode")
	}
	if req.KeyMode != dto.GithubKeyModeToken {
		project.Token, project.TokenUser = "", ""
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = githubDefaultTimeout
	}
	keep := req.KeepRecords
	if keep <= 0 {
		keep = githubDefaultKeepRecords
	}
	if keep > githubMaxKeepRecords {
		keep = githubMaxKeepRecords
	}
	project.Name = name
	project.CloneURL = cloneURL
	project.Repo = repoName
	project.Branch = branch
	project.ProjectPath = projectPath
	project.UpdateStrategy = req.UpdateStrategy
	if project.UpdateStrategy == "" {
		project.UpdateStrategy = dto.GithubStrategyFFOnly
	}
	project.Timeout = timeout
	project.KeepRecords = keep
	project.IgnorePaths = strings.Join(githubutil.ParseIgnorePaths(req.IgnorePaths), "\n")
	project.KeyMode = req.KeyMode
	project.Enabled = req.Enabled
	return nil
}

func githubProjectInfo(item model.GithubProject, setting dto.GithubSetting) dto.GithubProjectInfo {
	info := dto.GithubProjectInfo{
		ID:              item.ID,
		CreatedAt:       item.CreatedAt,
		Name:            item.Name,
		Repo:            item.Repo,
		CloneURL:        item.CloneURL,
		Branch:          item.Branch,
		ProjectPath:     item.ProjectPath,
		UpdateStrategy:  item.UpdateStrategy,
		Timeout:         item.Timeout,
		KeyMode:         item.KeyMode,
		KeyID:           item.KeyID,
		WebhookEnabled:  item.DomainMode != dto.GithubDomainNone,
		WebhookScriptID: item.WebhookScriptID,
		WebsiteID:       item.WebsiteID,
		RuntimeID:       item.RuntimeID,
		TokenUser:       item.TokenUser,
		HasToken:        item.Token != "",
		KeepRecords:     item.KeepRecords,
		IgnorePaths:     item.IgnorePaths,
		Enabled:         item.Enabled,
		LastStatus:      item.LastStatus,
		LastRunAt:       item.LastRunAt,
		LastCommit:      item.LastCommit,
		Running:         githubQueue.Running(item.ID),
	}
	if key, err := githubProjectKey(item, setting); err == nil && key.ID != 0 {
		info.KeyID = key.ID
		info.KeyName = key.Name
		info.PublicKey = key.PublicKey
	}
	target := githubWebhookTarget(item)
	info.WebhookURL = target.URL
	info.WebhookHTTPS = target.HTTPS
	return info
}

func githubValidCloneURL(cloneURL string) bool {
	if cloneURL == "" || strings.ContainsAny(cloneURL, " \t\r\n'\"`;|&$<>\\") || strings.HasPrefix(cloneURL, "-") {
		return false
	}
	switch {
	case strings.HasPrefix(cloneURL, "git@"):
		return strings.Contains(cloneURL, ":")
	case strings.HasPrefix(cloneURL, "ssh://"), strings.HasPrefix(cloneURL, "https://"):
		u, err := url.Parse(cloneURL)
		return err == nil && u.Host != "" && u.Path != "" && u.Path != "/"
	}
	return false
}

var githubBlockedPaths = []string{"/", "/bin", "/boot", "/dev", "/etc", "/lib", "/lib64", "/proc", "/root", "/run", "/sbin", "/sys", "/usr", "/var", "/tmp"}

func githubCleanProjectPath(raw string) (string, error) {
	p := strings.TrimSpace(raw)
	if p == "" || !strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\x00\r\n") {
		return "", buserr.New("ErrGithubInvalidPath")
	}
	p = filepath.Clean(p)
	for _, blocked := range githubBlockedPaths {
		if p == blocked {
			return "", buserr.New("ErrGithubInvalidPath")
		}
	}
	dataDir := filepath.Clean(global.Dir.DataDir)
	if dataDir != "" && dataDir != "." && (p == dataDir || p == filepath.Dir(dataDir) ||
		strings.HasPrefix(p, path.Join(dataDir, "db")) || strings.HasPrefix(p, path.Join(dataDir, "github"))) {
		return "", buserr.New("ErrGithubInvalidPath")
	}
	return p, nil
}

// ---------- settings ----------

func loadGithubSetting() dto.GithubSetting {
	get := func(key string) string {
		value, _ := settingRepo.GetValueByKey(key)
		return strings.TrimSpace(value)
	}
	atoi := func(key string, def int) int {
		if v, err := strconv.Atoi(get(key)); err == nil {
			return v
		}
		return def
	}
	setting := dto.GithubSetting{
		DefaultShell: get(githubSettingShell),
		LogRetention: atoi(githubSettingRetention, githubDefaultRetention),
		SharedKeyID:  uint(atoi(githubSettingSharedKeyID, 0)),
	}
	if setting.DefaultShell != "sh" {
		setting.DefaultShell = "bash"
	}
	if setting.LogRetention <= 0 {
		setting.LogRetention = githubDefaultRetention
	}
	return setting
}

func (g *GithubService) LoadSetting() (*dto.GithubSettingInfo, error) {
	address := githubPanelAddress()
	return &dto.GithubSettingInfo{
		GithubSetting: loadGithubSetting(),
		WebhookBase:   address.Base,
		BindDomain:    address.BindDomain,
		GitInstalled:  cmd.Which("git"),
	}, nil
}

func (g *GithubService) UpdateSetting(req dto.GithubSetting) error {
	if req.SharedKeyID != 0 {
		key, err := githubRepo.GetKey(repo.WithByID(req.SharedKeyID))
		if err != nil || key.Type != dto.GithubKeyTypeShared {
			return buserr.New("ErrGithubKeyNotFound")
		}
	}
	return settingRepo.UpdateValues(map[string]string{
		githubSettingShell:       req.DefaultShell,
		githubSettingRetention:   strconv.Itoa(req.LogRetention),
		githubSettingSharedKeyID: strconv.Itoa(int(req.SharedKeyID)),
	})
}
