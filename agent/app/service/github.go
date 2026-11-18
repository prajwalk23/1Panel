package service

// fork: GitHub section - projects, settings, webhook domains and the hook reverse proxy.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/url"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/buserr"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/cmd"
	"github.com/1Panel-dev/1Panel/agent/utils/encrypt"
	"github.com/1Panel-dev/1Panel/agent/utils/files"
	githubutil "github.com/1Panel-dev/1Panel/agent/utils/github"
)

var githubRepo = repo.NewIGithubRepo()

const (
	githubHookProxyName = "github-hook"
	githubHookPath      = "/github-hook/"

	githubSettingPort          = "GithubWebhookPort"
	githubSettingBind          = "GithubWebhookBind"
	githubSettingShell         = "GithubDefaultShell"
	githubSettingRetention     = "GithubLogRetention"
	githubSettingWebsiteID     = "GithubDefaultWebsiteID"
	githubSettingDomain        = "GithubDefaultDomain"
	githubSettingCustomBaseURL = "GithubCustomBaseURL"
	githubSettingSharedKeyID   = "GithubSharedKeyID"
	githubSettingHookSites     = "GithubHookSites"

	githubDefaultPort      = 19090
	githubDefaultTimeout   = 600
	githubDefaultRetention = 50
)

var githubUserPattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}\$?$`)

type GithubService struct{}

type IGithubService interface {
	SearchProjects(req dto.GithubProjectSearch) (int64, []dto.GithubProjectInfo, error)
	CreateProject(req dto.GithubProjectOperate) (*dto.GithubProjectSecret, error)
	UpdateProject(req dto.GithubProjectOperate) error
	DeleteProject(id uint) error
	UpdateProjectStatus(req dto.GithubProjectStatus) error
	RegenerateSecret(id uint) (*dto.GithubProjectSecret, error)
	LoadWebhookInfo(id uint) (*dto.GithubWebhookInfo, error)
	Deploy(id uint) error
	Cancel(id uint) error

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
	ListDomains() ([]dto.GithubDomainOption, error)
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

func (g *GithubService) CreateProject(req dto.GithubProjectOperate) (*dto.GithubProjectSecret, error) {
	if existing, _ := githubRepo.GetProject(repo.WithByName(strings.TrimSpace(req.Name))); existing.ID != 0 {
		return nil, buserr.New("ErrGithubNameExist")
	}
	project := model.GithubProject{}
	if err := githubApplyProject(&project, req); err != nil {
		return nil, err
	}
	setting := loadGithubSetting()

	createdKeyID := uint(0)
	if project.KeyMode == dto.GithubKeyModeProject {
		key, err := githubCreateKeyRecord(project.Name, dto.GithubKeyTypeProject, nil)
		if err != nil {
			return nil, err
		}
		project.KeyID = key.ID
		createdKeyID = key.ID
	}
	secret, err := githubNewSecret()
	if err != nil {
		return nil, err
	}
	if project.WebhookSecret, err = encrypt.StringEncrypt(secret); err != nil {
		return nil, err
	}

	if err := githubRepo.CreateProject(&project); err != nil {
		githubDropKey(createdKeyID)
		return nil, err
	}
	if err := githubSyncHookProxies(setting); err != nil {
		_ = githubRepo.DeleteProject(repo.WithByID(project.ID))
		githubDropKey(createdKeyID)
		return nil, err
	}
	info := githubProjectInfo(project, setting)
	return &dto.GithubProjectSecret{ID: project.ID, Secret: secret, WebhookURL: info.WebhookURL, PublicKey: info.PublicKey}, nil
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
		if oldKeyMode == dto.GithubKeyModeProject && oldKeyID != 0 {
			project.KeyID = oldKeyID
		} else {
			key, err := githubCreateKeyRecord(project.Name, dto.GithubKeyTypeProject, nil)
			if err != nil {
				return err
			}
			project.KeyID = key.ID
			createdKeyID = key.ID
		}
	}
	if err := githubRepo.SaveProject(&project); err != nil {
		githubDropKey(createdKeyID)
		return err
	}
	if oldKeyMode == dto.GithubKeyModeProject && oldKeyID != 0 && project.KeyID != oldKeyID {
		githubDropKey(oldKeyID)
	}
	return githubSyncHookProxies(loadGithubSetting())
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
	if project.KeyMode == dto.GithubKeyModeProject {
		githubDropKey(project.KeyID)
	}
	return githubSyncHookProxies(loadGithubSetting())
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
	if target.WebsiteID != 0 {
		if website, err := websiteRepo.GetFirst(repo.WithByID(target.WebsiteID)); err == nil {
			info.ProxyReady = githubHookProxyExists(website)
		}
	} else {
		info.ProxyReady = target.URL != ""
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
	runAs := strings.TrimSpace(req.RunAsUser)
	if runAs != "" {
		if !githubUserPattern.MatchString(runAs) {
			return buserr.New("ErrGithubInvalidUser")
		}
		if _, err := user.Lookup(runAs); err != nil {
			return buserr.New("ErrGithubInvalidUser")
		}
	}

	switch req.KeyMode {
	case dto.GithubKeyModeShared:
		setting := loadGithubSetting()
		if setting.SharedKeyID == 0 {
			return buserr.New("ErrGithubSharedKeyNotSet")
		}
		project.KeyID = 0
	case dto.GithubKeyModeCustom:
		key, err := githubRepo.GetKey(repo.WithByID(req.KeyID))
		if err != nil || key.Type != dto.GithubKeyTypeCustom {
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

	switch req.DomainMode {
	case dto.GithubDomainDefault:
		project.WebsiteID, project.Domain, project.CustomURL = 0, "", ""
	case dto.GithubDomainWebsite:
		website, err := websiteRepo.GetFirst(repo.WithByID(req.WebsiteID))
		if err != nil {
			return buserr.New("ErrGithubWebsiteNotFound")
		}
		domain := strings.TrimSpace(req.Domain)
		if domain != "" && githubFindDomain(website, domain) == nil {
			return buserr.New("ErrGithubDomainNotFound")
		}
		project.WebsiteID, project.Domain, project.CustomURL = website.ID, domain, ""
	case dto.GithubDomainCustom:
		customURL := strings.TrimRight(strings.TrimSpace(req.CustomURL), "/")
		if !githubValidBaseURL(customURL) {
			return buserr.New("ErrGithubInvalidURL")
		}
		project.WebsiteID, project.Domain, project.CustomURL = 0, "", customURL
	default:
		return buserr.New("ErrGithubInvalidURL")
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
	project.DeployCommand = strings.ReplaceAll(req.DeployCommand, "\r\n", "\n")
	project.UpdateStrategy = req.UpdateStrategy
	project.RunAsUser = runAs
	project.Timeout = timeout
	project.KeyMode = req.KeyMode
	project.DomainMode = req.DomainMode
	project.Enabled = req.Enabled
	return nil
}

func githubProjectInfo(item model.GithubProject, setting dto.GithubSetting) dto.GithubProjectInfo {
	info := dto.GithubProjectInfo{
		ID:             item.ID,
		CreatedAt:      item.CreatedAt,
		Name:           item.Name,
		Repo:           item.Repo,
		CloneURL:       item.CloneURL,
		Branch:         item.Branch,
		ProjectPath:    item.ProjectPath,
		DeployCommand:  item.DeployCommand,
		UpdateStrategy: item.UpdateStrategy,
		RunAsUser:      item.RunAsUser,
		Timeout:        item.Timeout,
		KeyMode:        item.KeyMode,
		KeyID:          item.KeyID,
		DomainMode:     item.DomainMode,
		WebsiteID:      item.WebsiteID,
		Domain:         item.Domain,
		CustomURL:      item.CustomURL,
		Enabled:        item.Enabled,
		LastStatus:     item.LastStatus,
		LastRunAt:      item.LastRunAt,
		LastCommit:     item.LastCommit,
		Running:        githubQueue.Running(item.ID),
	}
	if key, err := githubProjectKey(item, setting); err == nil && key.ID != 0 {
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
		WebhookPort:      atoi(githubSettingPort, githubDefaultPort),
		WebhookBind:      get(githubSettingBind),
		DefaultShell:     get(githubSettingShell),
		LogRetention:     atoi(githubSettingRetention, githubDefaultRetention),
		DefaultWebsiteID: uint(atoi(githubSettingWebsiteID, 0)),
		DefaultDomain:    get(githubSettingDomain),
		CustomBaseURL:    get(githubSettingCustomBaseURL),
		SharedKeyID:      uint(atoi(githubSettingSharedKeyID, 0)),
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
	defaultDomain := strings.TrimSpace(req.DefaultDomain)
	if req.DefaultWebsiteID != 0 {
		website, err := websiteRepo.GetFirst(repo.WithByID(req.DefaultWebsiteID))
		if err != nil {
			return buserr.New("ErrGithubWebsiteNotFound")
		}
		if defaultDomain != "" && githubFindDomain(website, defaultDomain) == nil {
			return buserr.New("ErrGithubDomainNotFound")
		}
	} else {
		defaultDomain = ""
	}
	customBase := strings.TrimRight(strings.TrimSpace(req.CustomBaseURL), "/")
	if customBase != "" && !githubValidBaseURL(customBase) {
		return buserr.New("ErrGithubInvalidURL")
	}

	values := map[string]string{
		githubSettingPort:          strconv.Itoa(req.WebhookPort),
		githubSettingBind:          bind,
		githubSettingShell:         req.DefaultShell,
		githubSettingRetention:     strconv.Itoa(req.LogRetention),
		githubSettingWebsiteID:     strconv.Itoa(int(req.DefaultWebsiteID)),
		githubSettingDomain:        defaultDomain,
		githubSettingCustomBaseURL: customBase,
		githubSettingSharedKeyID:   strconv.Itoa(int(req.SharedKeyID)),
	}
	if err := settingRepo.UpdateValues(values); err != nil {
		return err
	}
	setting := loadGithubSetting()
	if listenerChanged {
		if err := githubHook.restart(); err != nil {
			global.LOG.Errorf("[github] restart webhook listener failed: %v", err)
		}
	}
	return githubSyncHookProxies(setting)
}

func githubPortInUse(bind string, port int) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort(bind, strconv.Itoa(port)))
	if err != nil {
		return true
	}
	_ = ln.Close()
	return false
}

// ---------- domains and the hook reverse proxy ----------

func (g *GithubService) ListDomains() ([]dto.GithubDomainOption, error) {
	websites, err := websiteRepo.List()
	if err != nil {
		return nil, err
	}
	var options []dto.GithubDomainOption
	for _, website := range websites {
		hasProxy := githubHookProxyExists(website)
		https := website.Protocol == constant.ProtocolHTTPS
		if len(website.Domains) == 0 {
			options = append(options, dto.GithubDomainOption{
				WebsiteID: website.ID, Alias: website.Alias, PrimaryDomain: website.PrimaryDomain,
				Domain: website.PrimaryDomain, HTTPS: https, HookProxy: hasProxy,
			})
			continue
		}
		for _, domain := range website.Domains {
			options = append(options, dto.GithubDomainOption{
				WebsiteID: website.ID, Alias: website.Alias, PrimaryDomain: website.PrimaryDomain,
				Domain: domain.Domain, Port: domain.Port, HTTPS: https, HookProxy: hasProxy,
			})
		}
	}
	return options, nil
}

type githubTarget struct {
	URL       string
	HTTPS     bool
	WebsiteID uint
	Message   string
}

// githubResolveTarget works out the public webhook URL of a project.
func githubResolveTarget(project model.GithubProject, setting dto.GithubSetting) githubTarget {
	suffix := strings.TrimSuffix(githubHookPath, "/") + "/" + strconv.Itoa(int(project.ID))
	websiteID, domain, base := uint(0), "", ""
	switch project.DomainMode {
	case dto.GithubDomainWebsite:
		websiteID, domain = project.WebsiteID, project.Domain
	case dto.GithubDomainCustom:
		base = project.CustomURL
	default:
		if setting.DefaultWebsiteID != 0 {
			websiteID, domain = setting.DefaultWebsiteID, setting.DefaultDomain
		} else {
			base = setting.CustomBaseURL
		}
	}
	if base != "" {
		return githubTarget{URL: strings.TrimRight(base, "/") + suffix, HTTPS: strings.HasPrefix(base, "https://")}
	}
	if websiteID == 0 {
		return githubTarget{Message: "GithubNoDomain"}
	}
	website, err := websiteRepo.GetFirst(repo.WithByID(websiteID))
	if err != nil {
		return githubTarget{Message: "ErrGithubWebsiteNotFound"}
	}
	domains, _ := websiteDomainRepo.GetBy(websiteDomainRepo.WithWebsiteId(website.ID))
	website.Domains = domains
	host, port := website.PrimaryDomain, 0
	if item := githubFindDomain(website, domain); item != nil {
		host, port = item.Domain, item.Port
	} else if item := githubFindDomain(website, website.PrimaryDomain); item != nil {
		port = item.Port
	}
	https := website.Protocol == constant.ProtocolHTTPS
	scheme := "http"
	if https {
		scheme = "https"
	}
	hostPort := host
	if !https && port != 0 && port != 80 {
		hostPort = net.JoinHostPort(host, strconv.Itoa(port))
	}
	return githubTarget{URL: scheme + "://" + hostPort + suffix, HTTPS: https, WebsiteID: website.ID}
}

func githubFindDomain(website model.Website, domain string) *model.WebsiteDomain {
	if domain == "" {
		return nil
	}
	for i := range website.Domains {
		if strings.EqualFold(website.Domains[i].Domain, domain) {
			return &website.Domains[i]
		}
	}
	if len(website.Domains) == 0 {
		domains, _ := websiteDomainRepo.GetBy(websiteDomainRepo.WithWebsiteId(website.ID))
		for i := range domains {
			if strings.EqualFold(domains[i].Domain, domain) {
				return &domains[i]
			}
		}
	}
	return nil
}

func githubHookProxyFiles(website model.Website) (string, string) {
	dir := GetSitePath(website, SiteProxyDir)
	return path.Join(dir, githubHookProxyName+".conf"), path.Join(dir, githubHookProxyName+".bak")
}

func githubHookProxyExists(website model.Website) bool {
	conf, _ := githubHookProxyFiles(website)
	return files.NewFileOp().Stat(conf)
}

// githubWantedHookSites returns the websites that must carry the /github-hook/ proxy.
func githubWantedHookSites(setting dto.GithubSetting) map[uint]bool {
	wanted := map[uint]bool{}
	if setting.DefaultWebsiteID != 0 {
		wanted[setting.DefaultWebsiteID] = true
	}
	projects, _ := githubRepo.ListProjects()
	for _, project := range projects {
		if project.DomainMode == dto.GithubDomainWebsite && project.WebsiteID != 0 {
			wanted[project.WebsiteID] = true
		}
	}
	return wanted
}

func githubLoadHookSites() map[uint]bool {
	sites := map[uint]bool{}
	raw, _ := settingRepo.GetValueByKey(githubSettingHookSites)
	var ids []uint
	if raw != "" && json.Unmarshal([]byte(raw), &ids) == nil {
		for _, id := range ids {
			sites[id] = true
		}
	}
	return sites
}

func githubSaveHookSites(sites map[uint]bool) {
	ids := make([]uint, 0, len(sites))
	for id := range sites {
		ids = append(ids, id)
	}
	data, _ := json.Marshal(ids)
	if err := settingRepo.UpdateOrCreate(githubSettingHookSites, string(data)); err != nil {
		global.LOG.Errorf("[github] save hook sites failed: %v", err)
	}
}

// githubSyncHookProxies adds or updates the hook proxy on every wanted website
// and removes it from websites the panel added it to earlier but no longer needs.
func githubSyncHookProxies(setting dto.GithubSetting) error {
	wanted := githubWantedHookSites(setting)
	managed := githubLoadHookSites()
	var firstErr error
	for id := range wanted {
		if err := githubEnsureHookProxy(id, setting); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		managed[id] = true
	}
	for id := range managed {
		if wanted[id] {
			continue
		}
		if err := githubRemoveHookProxy(id); err != nil {
			global.LOG.Errorf("[github] remove hook proxy from website %d failed: %v", id, err)
			continue
		}
		delete(managed, id)
	}
	githubSaveHookSites(managed)
	return firstErr
}

func githubProxyPass(setting dto.GithubSetting) string {
	host := setting.WebhookBind
	if ip := net.ParseIP(host); ip == nil || ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(setting.WebhookPort))
}

func githubEnsureHookProxy(websiteID uint, setting dto.GithubSetting) error {
	website, err := websiteRepo.GetFirst(repo.WithByID(websiteID))
	if err != nil {
		return buserr.New("ErrGithubWebsiteNotFound")
	}
	if _, err := getAppInstallByKey(constant.AppOpenresty); err != nil {
		return buserr.New("ErrGithubNoOpenresty")
	}
	websiteService := NewIWebsiteService()
	conf, bak := githubHookProxyFiles(website)
	fileOp := files.NewFileOp()
	if !fileOp.Stat(conf) && fileOp.Stat(bak) {
		if err := websiteService.OperateProxy(request.WebsiteProxyConfig{ID: website.ID, Operate: "enable", Name: githubHookProxyName}); err != nil {
			return err
		}
	}
	operate := "create"
	if fileOp.Stat(conf) {
		operate = "edit"
	}
	return websiteService.OperateProxy(request.WebsiteProxyConfig{
		ID:        website.ID,
		Operate:   operate,
		Enable:    true,
		Name:      githubHookProxyName,
		Modifier:  "^~",
		Match:     githubHookPath,
		ProxyPass: githubProxyPass(setting),
		ProxyHost: "$host",
	})
}

func githubRemoveHookProxy(websiteID uint) error {
	website, err := websiteRepo.GetFirst(repo.WithByID(websiteID))
	if err != nil {
		return nil
	}
	conf, bak := githubHookProxyFiles(website)
	fileOp := files.NewFileOp()
	if !fileOp.Stat(conf) && !fileOp.Stat(bak) {
		return nil
	}
	return NewIWebsiteService().DeleteProxy(request.WebsiteProxyDel{ID: website.ID, Name: githubHookProxyName})
}
