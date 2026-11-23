package service

// fork: GitHub source support for runtimes. A runtime can clone its code
// directory from a repository and pull updates on restart/update. It reuses the
// SSH deploy keys, known_hosts and clone helpers of the GitHub section.

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
	"github.com/1Panel-dev/1Panel/agent/app/task"
	"github.com/1Panel-dev/1Panel/agent/buserr"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/cmd"
	githubutil "github.com/1Panel-dev/1Panel/agent/utils/github"
)

const runtimeGitDefaultTimeout = 600

// runtimeGitEnabled reports whether the runtime clones its code from GitHub.
func runtimeGitEnabled(runtime *model.Runtime) bool {
	return runtime != nil && runtime.GitMode != "" && runtime.GitMode != dto.RuntimeGitModeNone
}

// runtimeGitRequested reports whether a create request asks for a git source.
func runtimeGitRequested(create request.RuntimeCreate) bool {
	mode := strings.TrimSpace(create.GitMode)
	return mode != "" && mode != dto.RuntimeGitModeNone
}

// applyRuntimeGit fills the git source fields of a runtime from a create request.
func applyRuntimeGit(runtime *model.Runtime, mode, cloneURL, branch string, keyID uint, strategy string) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = dto.RuntimeGitModeNone
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch = "main"
	}
	if strategy != dto.GithubStrategyReset {
		strategy = dto.GithubStrategyFFOnly
	}
	runtime.GitMode = mode
	runtime.GitCloneURL = strings.TrimSpace(cloneURL)
	runtime.GitBranch = branch
	runtime.GitKeyID = keyID
	runtime.GitUpdateStrategy = strategy
	if mode == dto.RuntimeGitModeNone {
		runtime.GitCloneURL = ""
		runtime.GitKeyID = 0
	}
}

// applyRuntimeGitFromCreate copies the git source of a create/update request.
func applyRuntimeGitFromCreate(runtime *model.Runtime, create request.RuntimeCreate) {
	applyRuntimeGit(runtime, create.GitMode, create.GitCloneURL, create.GitBranch, create.GitKeyID, create.GitUpdateStrategy)
}

// applyRuntimeGitFromUpdate copies the git source of an update request.
func applyRuntimeGitFromUpdate(runtime *model.Runtime, req request.RuntimeUpdate) {
	applyRuntimeGit(runtime, req.GitMode, req.GitCloneURL, req.GitBranch, req.GitKeyID, req.GitUpdateStrategy)
}

// validateRuntimeGit rejects an unusable git source before the task starts.
func validateRuntimeGit(create request.RuntimeCreate) error {
	return validateRuntimeGitFields(create.GitMode, create.GitCloneURL, create.GitBranch, create.CodeDir)
}

func validateRuntimeGitFields(mode, cloneURL, branch, codeDir string) error {
	if err := validateGitSource(mode, cloneURL, branch); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(mode)
	if trimmed == "" || trimmed == dto.RuntimeGitModeNone {
		return nil
	}
	if strings.TrimSpace(codeDir) == "" {
		return buserr.New("ErrPathNotFound")
	}
	return nil
}

// validateGitSource rejects an unusable git source before any work starts.
func validateGitSource(mode, cloneURL, branch string) error {
	mode = strings.TrimSpace(mode)
	if mode == "" || mode == dto.RuntimeGitModeNone {
		return nil
	}
	if mode != dto.RuntimeGitModeGit {
		return buserr.New("ErrGithubInvalidKeyMode")
	}
	if !githubValidCloneURL(strings.TrimSpace(cloneURL)) {
		return buserr.New("ErrGithubInvalidCloneURL")
	}
	if branch = strings.TrimSpace(branch); branch == "" {
		branch = "main"
	}
	if !githubutil.ValidBranch(branch) {
		return buserr.New("ErrGithubInvalidBranch")
	}
	return nil
}

// applyWebsiteGit copies the git source of a create request onto a website.
func applyWebsiteGit(website *model.Website, create request.WebsiteCreate) {
	mode := strings.TrimSpace(create.GitMode)
	if mode == "" {
		mode = dto.RuntimeGitModeNone
	}
	branch := strings.TrimSpace(create.GitBranch)
	if branch == "" {
		branch = "main"
	}
	strategy := create.GitUpdateStrategy
	if strategy != dto.GithubStrategyReset {
		strategy = dto.GithubStrategyFFOnly
	}
	website.GitMode = mode
	website.GitCloneURL = strings.TrimSpace(create.GitCloneURL)
	website.GitBranch = branch
	website.GitKeyID = create.GitKeyID
	website.GitUpdateStrategy = strategy
	if mode == dto.RuntimeGitModeNone {
		website.GitCloneURL = ""
		website.GitKeyID = 0
	}
}

// resolveGitKey returns the SSH key for a clone: a specific key wins, otherwise
// the shared key of the GitHub settings is used. Public (https) clones need none.
func resolveGitKey(keyID uint, cloneURL string) (model.GithubKey, error) {
	if keyID != 0 {
		return githubRepo.GetKey(repo.WithByID(keyID))
	}
	setting := loadGithubSetting()
	if setting.SharedKeyID != 0 {
		return githubRepo.GetKey(repo.WithByID(setting.SharedKeyID))
	}
	if githubutil.IsSSHCloneURL(cloneURL) {
		return model.GithubKey{}, buserr.New("ErrGithubKeyRequired")
	}
	return model.GithubKey{}, nil
}

// resolveRuntimeGitKey returns the SSH key used to reach the runtime repository.
func resolveRuntimeGitKey(runtime *model.Runtime) (model.GithubKey, error) {
	return resolveGitKey(runtime.GitKeyID, runtime.GitCloneURL)
}

func runtimeGitEnv(key model.GithubKey) []string {
	env := []string{"GIT_TERMINAL_PROMPT=0"}
	if key.ID != 0 {
		env = append(env, "GIT_SSH_COMMAND="+githubutil.SSHCommand(key.KeyPath, githubKnownHosts()))
	}
	return env
}

// gitRun runs one git command. Output goes to the task when there is one, else
// to logPath when given.
func gitRun(ctx context.Context, logPath string, t *task.Task, workDir string, env []string, args ...string) (string, error) {
	full := append([]string{"-c", "safe.directory=*"}, args...)
	opts := []cmd.Option{
		cmd.WithContext(ctx),
		cmd.WithTimeout(runtimeGitDefaultTimeout * time.Second),
		cmd.WithWorkDir(workDir),
		cmd.WithEnv(env...),
	}
	if t != nil {
		opts = append(opts, cmd.WithTask(*t))
	} else if logPath != "" {
		opts = append(opts, cmd.WithOutputFile(logPath))
	}
	mgr := cmd.NewCommandMgr(opts...)
	out, err := mgr.RunWithStdout("git", full...)
	return strings.TrimSpace(out), err
}

// gitSyncCheckout clones dir from cloneURL the first time and updates the
// checkout on later runs. It returns the resulting HEAD.
func gitSyncCheckout(ctx context.Context, logPath string, t *task.Task, dir, cloneURL, branch, strategy string, key model.GithubKey) (string, error) {
	if !cmd.Which("git") {
		return "", buserr.New("ErrGithubGitNotFound")
	}
	env := runtimeGitEnv(key)
	if branch == "" {
		branch = "main"
	}

	if _, statErr := os.Stat(path.Join(dir, ".git")); statErr != nil {
		entries, readErr := os.ReadDir(dir)
		if readErr == nil && len(entries) != 0 {
			return "", buserr.WithName("ErrGithubPathNotEmpty", dir)
		}
		parent := path.Dir(dir)
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return "", err
		}
		if _, err := gitRun(ctx, logPath, t, parent, env, "clone", "--branch", branch, "--single-branch", "--", cloneURL, dir); err != nil {
			return "", err
		}
	} else {
		if current, _ := gitRun(ctx, logPath, t, dir, env, "remote", "get-url", "origin"); current != cloneURL {
			if _, err := gitRun(ctx, logPath, t, dir, env, "remote", "set-url", "origin", cloneURL); err != nil {
				return "", err
			}
		}
		refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch)
		if _, err := gitRun(ctx, logPath, t, dir, env, "fetch", "--prune", "origin", refspec); err != nil {
			return "", err
		}
		remoteRef := "origin/" + branch
		if current, _ := gitRun(ctx, logPath, t, dir, env, "rev-parse", "--abbrev-ref", "HEAD"); current != branch {
			if _, err := gitRun(ctx, logPath, t, dir, env, "checkout", "-B", branch, remoteRef); err != nil {
				return "", err
			}
		}
		if strategy == dto.GithubStrategyReset {
			if _, err := gitRun(ctx, logPath, t, dir, env, "reset", "--hard", remoteRef); err != nil {
				return "", err
			}
		} else {
			if _, err := gitRun(ctx, logPath, t, dir, env, "merge", "--ff-only", remoteRef); err != nil {
				return "", err
			}
		}
	}

	head, err := gitRun(ctx, logPath, t, dir, env, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	if !githubutil.ValidSHA(head) {
		return "", nil
	}
	return head, nil
}

// runtimeGitSync clones the repository into the runtime code directory the first
// time and updates the checkout on later runs. It returns the resulting HEAD.
func runtimeGitSync(ctx context.Context, runtime *model.Runtime, t *task.Task) (string, error) {
	key, err := resolveRuntimeGitKey(runtime)
	if err != nil {
		return "", err
	}
	return gitSyncCheckout(ctx, runtime.GetLogPath(), t, runtime.CodeDir, runtime.GitCloneURL, runtime.GitBranch, runtime.GitUpdateStrategy, key)
}

// websiteGitEnabled reports whether a PHP website clones its code from GitHub.
func websiteGitEnabled(website *model.Website) bool {
	return website != nil && website.GitMode != "" && website.GitMode != dto.RuntimeGitModeNone
}

// websiteGitClone clones or updates the website repository into its index dir.
func websiteGitClone(website *model.Website) error {
	key, err := resolveGitKey(website.GitKeyID, website.GitCloneURL)
	if err != nil {
		return err
	}
	dir := GetSitePath(*website, SiteIndexDir)
	commit, err := gitSyncCheckout(context.Background(), "", nil, dir, website.GitCloneURL, website.GitBranch, website.GitUpdateStrategy, key)
	if err != nil {
		return err
	}
	global.LOG.Infof("website [%s] git clone ok, HEAD %s", website.Alias, commit)
	return nil
}

// runtimeGitCloneRepo is the create-task entry point: it fails the task on error.
func runtimeGitCloneRepo(ctx context.Context, runtime *model.Runtime, t *task.Task) (string, error) {
	return runtimeGitSync(ctx, runtime, t)
}

// runtimeGitPullIfEnabled pulls the latest code before a start/restart. It is
// best effort: a failure is written to the runtime log but does not stop the
// container from starting on the code that is already on disk.
func runtimeGitPullIfEnabled(runtime *model.Runtime) error {
	if !runtimeGitEnabled(runtime) {
		return nil
	}
	commit, err := runtimeGitSync(context.Background(), runtime, nil)
	if err != nil {
		global.LOG.Errorf("runtime [%s] git pull failed: %v", runtime.Name, err)
		return err
	}
	if commit != "" {
		global.LOG.Infof("runtime [%s] git pull ok, HEAD %s", runtime.Name, commit)
	}
	return nil
}
