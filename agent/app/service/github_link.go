package service

// fork: a website or a runtime that takes its code from a repository owns one GitHub project.
// The project carries the scripts, webhook, deploy history and rollback of the Git Manager.
// Projects of sites and runtimes made before the Git Manager existed are created on first use.

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/buserr"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/encrypt"
	githubutil "github.com/1Panel-dev/1Panel/agent/utils/github"
)

// Authentication modes of a create request.
const (
	githubAuthPublic = "public"
	githubAuthToken  = "token"
	githubAuthSSH    = "ssh"
)

// githubLinkKeyMode maps what the create form chose to the key mode of a project.
func githubLinkKeyMode(authMode string, keyID uint, cloneURL string) string {
	switch authMode {
	case githubAuthToken:
		return dto.GithubKeyModeToken
	case githubAuthPublic:
		return dto.GithubKeyModeNone
	}
	if keyID != 0 {
		return dto.GithubKeyModeCustom
	}
	if githubutil.IsSSHCloneURL(cloneURL) || authMode == githubAuthSSH {
		return dto.GithubKeyModeShared
	}
	return dto.GithubKeyModeNone
}

// gitCredentials returns the SSH key or the extra git environment (access token) of a clone.
func gitCredentials(authMode string, keyID uint, cloneURL, tokenUser, token string) (model.GithubKey, []string, error) {
	if authMode == githubAuthToken {
		if strings.TrimSpace(token) == "" {
			return model.GithubKey{}, nil, buserr.New("ErrGithubTokenRequired")
		}
		return model.GithubKey{}, githubutil.TokenEnv(strings.TrimSpace(tokenUser), strings.TrimSpace(token)), nil
	}
	key, err := resolveGitKey(keyID, cloneURL)
	return key, nil, err
}

type githubLinkSource struct {
	WebsiteID uint
	RuntimeID uint
	Name      string
	Path      string
	CloneURL  string
	Branch    string
	AuthMode  string
	KeyID     uint
	TokenUser string
	Token     string // plain text, empty when unknown
	Commit    string
}

func githubUniqueProjectName(base string) string {
	name := base
	for i := 2; ; i++ {
		if existing, _ := githubRepo.GetProject(repo.WithByName(name)); existing.ID == 0 {
			return name
		}
		name = fmt.Sprintf("%s-%d", base, i)
	}
}

// githubCreateLinkedProject stores the project of a website or runtime and seeds its first
// deploy record, so the Deployments tab starts with the version that was cloned.
func githubCreateLinkedProject(src githubLinkSource) (model.GithubProject, error) {
	project := model.GithubProject{
		Name:           githubUniqueProjectName(src.Name),
		Repo:           githubutil.RepoFromCloneURL(src.CloneURL),
		CloneURL:       src.CloneURL,
		Branch:         src.Branch,
		ProjectPath:    src.Path,
		UpdateStrategy: dto.GithubStrategyFFOnly,
		Timeout:        githubDefaultTimeout,
		KeyMode:        githubLinkKeyMode(src.AuthMode, src.KeyID, src.CloneURL),
		KeyID:          src.KeyID,
		DomainMode:     dto.GithubDomainNone,
		WebsiteID:      src.WebsiteID,
		RuntimeID:      src.RuntimeID,
		Enabled:        true,
		KeepRecords:    githubDefaultKeepRecords,
	}
	if project.KeyMode == dto.GithubKeyModeToken {
		project.KeyID = 0
		project.TokenUser = strings.TrimSpace(src.TokenUser)
		if token := strings.TrimSpace(src.Token); token != "" {
			encrypted, err := encrypt.StringEncrypt(token)
			if err != nil {
				return project, err
			}
			project.Token = encrypted
		}
	} else if project.KeyMode == dto.GithubKeyModeShared || project.KeyMode == dto.GithubKeyModeNone {
		project.KeyID = 0
	}
	now := time.Now()
	if src.Commit != "" {
		project.LastStatus = dto.GithubStatusSuccess
		project.LastRunAt = now
		project.LastCommit = src.Commit
	}
	if err := githubRepo.CreateProject(&project); err != nil {
		return project, err
	}
	if src.Commit != "" {
		logItem := model.GithubDeployLog{
			ProjectID:  project.ID,
			Trigger:    dto.GithubTriggerCreate,
			Ref:        "refs/heads/" + project.Branch,
			Commit:     src.Commit,
			Status:     dto.GithubStatusSuccess,
			StartedAt:  now,
			FinishedAt: now,
		}
		if env, err := githubProjectEnv(project, loadGithubSetting()); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			if out, err := githubGitQuiet(ctx, project.ProjectPath, env, "log", "-1", "--format=%an%x1f%s", src.Commit); err == nil {
				if parts := strings.SplitN(out, "\x1f", 2); len(parts) == 2 {
					logItem.CommitAuthor, logItem.CommitMessage = githubTrim(parts[0], 100), githubTrim(parts[1], 500)
				}
			}
			cancel()
		}
		_ = githubRepo.CreateLog(&logItem)
	}
	return project, nil
}

func githubHeadCommit(dir string) string {
	if _, err := os.Stat(path.Join(dir, ".git")); err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := githubGitQuiet(ctx, dir, []string{"GIT_TERMINAL_PROMPT=0"}, "rev-parse", "HEAD")
	if err != nil || !githubutil.ValidSHA(out) {
		return ""
	}
	return out
}

// githubLinkWebsite creates the project of a website that was created from a repository.
func githubLinkWebsite(website *model.Website) {
	if !websiteGitEnabled(website) {
		return
	}
	dir := GetSitePath(*website, SiteIndexDir)
	if _, err := githubCreateLinkedProject(githubLinkSource{
		WebsiteID: website.ID,
		Name:      "website-" + website.Alias,
		Path:      dir,
		CloneURL:  website.GitCloneURL,
		Branch:    website.GitBranch,
		AuthMode:  website.GitAuthMode,
		KeyID:     website.GitKeyID,
		TokenUser: website.GitTokenUser,
		Token:     website.GitToken,
		Commit:    githubHeadCommit(dir),
	}); err != nil {
		global.LOG.Errorf("[github] link website %s failed: %v", website.Alias, err)
	}
}

// githubLinkRuntime creates the project of a runtime that was created from a repository.
func githubLinkRuntime(runtime *model.Runtime, create request.RuntimeCreate) {
	if !runtimeGitEnabled(runtime) {
		return
	}
	if _, err := githubCreateLinkedProject(githubLinkSource{
		RuntimeID: runtime.ID,
		Name:      "runtime-" + runtime.Name,
		Path:      runtime.CodeDir,
		CloneURL:  runtime.GitCloneURL,
		Branch:    runtime.GitBranch,
		AuthMode:  create.GitAuthMode,
		KeyID:     runtime.GitKeyID,
		TokenUser: create.GitTokenUser,
		Token:     create.GitToken,
		Commit:    githubHeadCommit(runtime.CodeDir),
	}); err != nil {
		global.LOG.Errorf("[github] link runtime %s failed: %v", runtime.Name, err)
	}
}

// githubDeleteLinked removes the project of a website or runtime that is being deleted.
func githubDeleteLinked(websiteID, runtimeID uint) {
	var opt repo.DBOption
	switch {
	case websiteID != 0:
		opt = githubRepo.WithByWebsiteID(websiteID)
	case runtimeID != 0:
		opt = githubRepo.WithByRuntimeID(runtimeID)
	default:
		return
	}
	projects, err := githubRepo.ListProjects(opt)
	if err != nil {
		return
	}
	for _, project := range projects {
		if err := githubRemoveProject(project); err != nil {
			global.LOG.Errorf("[github] remove linked project %s failed: %v", project.Name, err)
		}
	}
}

// LoadLinked returns the project of a website or a runtime.
func (g *GithubService) LoadLinked(req dto.GithubLinkedSearch) (*dto.GithubProjectInfo, error) {
	var opt repo.DBOption
	switch {
	case req.WebsiteID != 0:
		opt = githubRepo.WithByWebsiteID(req.WebsiteID)
	case req.RuntimeID != 0:
		opt = githubRepo.WithByRuntimeID(req.RuntimeID)
	default:
		return nil, buserr.New("ErrRecordNotFound")
	}
	project, err := githubRepo.GetProject(opt)
	if err != nil {
		project, err = githubLegacyProject(req)
		if err != nil {
			return nil, err
		}
	}
	info := githubProjectInfo(project, loadGithubSetting())
	return &info, nil
}

// githubLegacyProject creates the project of a site or runtime made before the Git Manager.
func githubLegacyProject(req dto.GithubLinkedSearch) (model.GithubProject, error) {
	if req.WebsiteID != 0 {
		website, err := websiteRepo.GetFirst(repo.WithByID(req.WebsiteID))
		if err != nil || !websiteGitEnabled(&website) {
			return model.GithubProject{}, buserr.New("ErrRecordNotFound")
		}
		dir := GetSitePath(website, SiteIndexDir)
		return githubCreateLinkedProject(githubLinkSource{
			WebsiteID: website.ID,
			Name:      "website-" + website.Alias,
			Path:      dir,
			CloneURL:  website.GitCloneURL,
			Branch:    website.GitBranch,
			KeyID:     website.GitKeyID,
			Commit:    githubHeadCommit(dir),
		})
	}
	runtime, err := runtimeRepo.GetFirst(context.Background(), repo.WithByID(req.RuntimeID))
	if err != nil || !runtimeGitEnabled(runtime) {
		return model.GithubProject{}, buserr.New("ErrRecordNotFound")
	}
	return githubCreateLinkedProject(githubLinkSource{
		RuntimeID: runtime.ID,
		Name:      "runtime-" + runtime.Name,
		Path:      runtime.CodeDir,
		CloneURL:  runtime.GitCloneURL,
		Branch:    runtime.GitBranch,
		KeyID:     runtime.GitKeyID,
		Commit:    githubHeadCommit(runtime.CodeDir),
	})
}
