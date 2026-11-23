package service

// fork: GitHub section - projects, settings and the webhook address.

import (
	"crypto/rand"
	"encoding/hex"
	"net"
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
	githubHookPath = "/github-hook/"

	githubSettingPort          = "GithubWebhookPort"
	githubSettingBind          = "GithubWebhookBind"
	githubSettingShell         = "GithubDefaultShell"
	githubSettingRetention     = "GithubLogRetention"
	githubSettingWebhookDomain = "GithubCustomBaseURL" // typed webhook domain (key kept from the first schema)
	githubSettingSharedKeyID   = "GithubSharedKeyID"

	githubDefaultPort      = 19090
	githubDefaultTimeout   = 600
	githubDefaultRetention = 50
)

var githubUserPattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}\$?$`)

type GithubService struct{}

type IGithubService interface {
	SearchProjects(req dto.GithubProjectSearch) (int64, []dto.GithubProjectInfo, error)
	CreateProject(req dto.GithubProjectOperate) (*dto.GithubProjectInfo, error)
	UpdateProject(req dto.GithubProjectOperate) error
	DeleteProject(id uint) error
	UpdateProjectStatus(req dto.GithubProjectStatus) error
	RegenerateSecret(id uint) (*dto.GithubProjectSecret, error)
	LoadWebhookInfo(id uint) (*dto.GithubWebhookInfo, error)
	ConfigureWebhook(req dto.GithubWebhookConfig) (*dto.GithubProjectSecret, error)
	Deploy(req dto.GithubProjectDeploy) error
	Cancel(id uint) error

	ListScripts(req dto.GithubScriptList) ([]dto.GithubScriptInfo, error)
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
	total, items, err := githubRepo.PageProjects(req.Page, req.PageSize, repo.WithByLikeName(req.Info), repo.WithOrderDesc("created_at"))
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
	// The webhook is configured later, when editing the project.
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

func (g *GithubService) RegenerateSecret(id uint) (*dto.GithubProjectSecret, error) {
	project, err := githubRepo.GetProject(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New("ErrRecordNotFound")
	}
	secret, err := githubNewSecret()
	if err != nil {
		return nil, err
	}
	encrypted, err := encrypt.StringEncrypt(secret)
	if err != nil {
		return nil, err
	}
	if err := githubRepo.UpdateProject(id, map[string]interface{}{"webhook_secret": encrypted}); err != nil {
		return nil, err
	}
	info := githubProjectInfo(project, loadGithubSetting())
	return &dto.GithubProjectSecret{ID: id, Secret: secret, WebhookURL: info.WebhookURL, PublicKey: info.PublicKey}, nil
}

func (g *GithubService) LoadWebhookInfo(id uint) (*dto.GithubWebhookInfo, error) {
	project, err := githubRepo.GetProject(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New("ErrRecordNotFound")
	}
	setting := loadGithubSetting()
	target := githubResolveTarget(project, setting)
	info := &dto.GithubWebhookInfo{
		Enabled:     project.DomainMode != dto.GithubDomainNone,
		ScriptID:    project.WebhookScriptID,
		URL:         target.URL,
		HTTPS:       target.HTTPS,
		ContentType: "application/json",
		Events:      "push",
		Message:     target.Message,
	}
	running, addr, _ := githubHook.status()
	if running {
		info.Listener = addr
	}
	info.ProxyPass = githubProxyPass(setting)
	if script, err := githubResolveScript(&project, 0); err == nil {
		info.ScriptID, info.ScriptName = script.ID, script.Name
	}
	return info, nil
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
	case dto.GithubKeyModeProject:
		// key is created or kept by the caller
	default:
		return buserr.New("ErrGithubInvalidKeyMode")
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = githubDefaultTimeout
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
	target := githubResolveTarget(item, setting)
	info.WebhookURL = target.URL
	info.WebhookHTTPS = target.HTTPS
	return info
}

func githubNewSecret() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
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

func githubValidBaseURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
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
		WebhookPort:   atoi(githubSettingPort, githubDefaultPort),
		WebhookBind:   get(githubSettingBind),
		DefaultShell:  get(githubSettingShell),
		LogRetention:  atoi(githubSettingRetention, githubDefaultRetention),
		WebhookDomain: get(githubSettingWebhookDomain),
		SharedKeyID:   uint(atoi(githubSettingSharedKeyID, 0)),
	}
	if setting.WebhookPort <= 0 || setting.WebhookPort > 65535 {
		setting.WebhookPort = githubDefaultPort
	}
	if setting.WebhookBind == "" {
		setting.WebhookBind = "127.0.0.1"
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
	setting := loadGithubSetting()
	running, addr, listenErr := githubHook.status()
	return &dto.GithubSettingInfo{
		GithubSetting:   setting,
		ListenerRunning: running,
		ListenerAddr:    addr,
		ListenerError:   listenErr,
		GitInstalled:    cmd.Which("git"),
	}, nil
}

func (g *GithubService) UpdateSetting(req dto.GithubSetting) error {
	old := loadGithubSetting()
	bind := strings.TrimSpace(req.WebhookBind)
	if net.ParseIP(bind) == nil {
		return buserr.New("ErrGithubInvalidBind")
	}
	listenerChanged := req.WebhookPort != old.WebhookPort || bind != old.WebhookBind
	if listenerChanged && githubPortInUse(bind, req.WebhookPort) {
		return buserr.WithName("ErrGithubPortInUse", strconv.Itoa(req.WebhookPort))
	}
	if req.SharedKeyID != 0 {
		key, err := githubRepo.GetKey(repo.WithByID(req.SharedKeyID))
		if err != nil || key.Type != dto.GithubKeyTypeShared {
			return buserr.New("ErrGithubKeyNotFound")
		}
	}
	webhookDomain, err := githubNormalizeWebhookDomain(req.WebhookDomain)
	if err != nil {
		return err
	}
	if webhookDomain != "" && webhookDomain != old.WebhookDomain {
		if err := githubCheckDomainFree(webhookDomain); err != nil {
			return err
		}
	}

	values := map[string]string{
		githubSettingPort:          strconv.Itoa(req.WebhookPort),
		githubSettingBind:          bind,
		githubSettingShell:         req.DefaultShell,
		githubSettingRetention:     strconv.Itoa(req.LogRetention),
		githubSettingWebhookDomain: webhookDomain,
		githubSettingSharedKeyID:   strconv.Itoa(int(req.SharedKeyID)),
	}
	if err := settingRepo.UpdateValues(values); err != nil {
		return err
	}
	if listenerChanged {
		if err := githubHook.restart(); err != nil {
			global.LOG.Errorf("[github] restart webhook listener failed: %v", err)
		}
	}
	return nil
}

func githubPortInUse(bind string, port int) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort(bind, strconv.Itoa(port)))
	if err != nil {
		return true
	}
	_ = ln.Close()
	return false
}

// ---------- webhook address ----------

type githubTarget struct {
	URL     string
	HTTPS   bool
	Message string
}

// githubNormalizeWebhookDomain turns what the user typed (hooks.example.com or
// https://hooks.example.com:8443) into scheme://host[:port]. Empty stays empty.
func githubNormalizeWebhookDomain(raw string) (string, error) {
	value := strings.TrimRight(strings.TrimSpace(raw), "/")
	if value == "" {
		return "", nil
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	u, err := url.Parse(value)
	if err != nil || !githubValidBaseURL(value) || u.User != nil || (u.Path != "" && u.Path != "/") {
		return "", buserr.New("ErrGithubInvalidURL")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || strings.ContainsAny(host, " _") {
		return "", buserr.New("ErrGithubInvalidURL")
	}
	if u.Port() != "" {
		host = net.JoinHostPort(host, u.Port())
	}
	return u.Scheme + "://" + host, nil
}

// githubCheckDomainFree fails when a 1Panel website already serves this host, because
// a website with that name would answer the webhook requests instead of the listener.
func githubCheckDomainFree(base string) error {
	u, err := url.Parse(base)
	if err != nil {
		return buserr.New("ErrGithubInvalidURL")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || net.ParseIP(host) != nil {
		return nil
	}
	websites, err := websiteRepo.List()
	if err != nil {
		return err
	}
	for _, website := range websites {
		if strings.EqualFold(website.PrimaryDomain, host) {
			return buserr.WithName("ErrGithubDomainIsWebsite", host)
		}
		for _, item := range website.Domains {
			if strings.EqualFold(item.Domain, host) {
				return buserr.WithName("ErrGithubDomainIsWebsite", host)
			}
		}
	}
	return nil
}

// githubResolveTarget works out the public webhook URL of a project. The domain
// always comes from the GitHub settings.
func githubResolveTarget(project model.GithubProject, setting dto.GithubSetting) githubTarget {
	if project.DomainMode == dto.GithubDomainNone {
		return githubTarget{}
	}
	if setting.WebhookDomain == "" {
		return githubTarget{Message: "GithubNoDomain"}
	}
	suffix := strings.TrimSuffix(githubHookPath, "/") + "/" + strconv.Itoa(int(project.ID))
	return githubTarget{URL: setting.WebhookDomain + suffix, HTTPS: strings.HasPrefix(setting.WebhookDomain, "https://")}
}

// githubProxyPass is where a reverse proxy must send /github-hook/ requests.
func githubProxyPass(setting dto.GithubSetting) string {
	host := setting.WebhookBind
	if ip := net.ParseIP(host); ip == nil || ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(setting.WebhookPort))
}
