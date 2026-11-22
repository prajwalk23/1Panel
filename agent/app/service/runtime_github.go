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
	if strings.TrimSpace(codeDir) == "" {
		return buserr.New("ErrPathNotFound")
	}
	return nil
}

// resolveRuntimeGitKey returns the SSH key used to reach the repository. A key
// chosen on the runtime wins; otherwise the shared key of the GitHub settings is
// used. Public (https) clones do not need a key.
func resolveRuntimeGitKey(runtime *model.Runtime) (model.GithubKey, error) {
	if runtime.GitKeyID != 0 {
		return githubRepo.GetKey(repo.WithByID(runtime.GitKeyID))
	}
	setting := loadGithubSetting()
	if setting.SharedKeyID != 0 {
		return githubRepo.GetKey(repo.WithByID(setting.SharedKeyID))
	}
	if githubutil.IsSSHCloneURL(runtime.GitCloneURL) {
		return model.GithubKey{}, buserr.New("ErrGithubKeyRequired")
	}
	return model.GithubKey{}, nil
}

func runtimeGitEnv(key model.GithubKey) []string {
	env := []string{"GIT_TERMINAL_PROMPT=0"}
	if key.ID != 0 {
		env = append(env, "GIT_SSH_COMMAND="+githubutil.SSHCommand(key.KeyPath, githubKnownHosts()))
	}
	return env
}

// runtimeGitRun runs one git command. Output goes to the task when there is one,
// otherwise it is appended to the runtime log file.
func runtimeGitRun(ctx context.Context, runtime *model.Runtime, t *task.Task, workDir string, env []string, args ...string) (string, error) {
	full := append([]string{"-c", "safe.directory=*"}, args...)
	opts := []cmd.Option{
		cmd.WithContext(ctx),
		cmd.WithTimeout(runtimeGitDefaultTimeout * time.Second),
		cmd.WithWorkDir(workDir),
		cmd.WithEnv(env...),
	}
	if t != nil {
		opts = append(opts, cmd.WithTask(*t))
	} else {
		opts = append(opts, cmd.WithOutputFile(runtime.GetLogPath()))
	}
	mgr := cmd.NewCommandMgr(opts...)
	out, err := mgr.RunWithStdout("git", full...)
	return strings.TrimSpace(out), err
}

// runtimeGitSync clones the repository into the runtime code directory the first
// time and updates the checkout on later runs. It returns the resulting HEAD.
func runtimeGitSync(ctx context.Context, runtime *model.Runtime, t *task.Task) (string, error) {
	if !cmd.Which("git") {
		return "", buserr.New("ErrGithubGitNotFound")
	}
	key, err := resolveRuntimeGitKey(runtime)
	if err != nil {
		return "", err
	}
	env := runtimeGitEnv(key)
	dir := runtime.CodeDir
	branch := runtime.GitBranch
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
		if _, err := runtimeGitRun(ctx, runtime, t, parent, env, "clone", "--branch", branch, "--single-branch", "--", runtime.GitCloneURL, dir); err != nil {
			return "", err
		}
	} else {
		if current, _ := runtimeGitRun(ctx, runtime, t, dir, env, "remote", "get-url", "origin"); current != runtime.GitCloneURL {
			if _, err := runtimeGitRun(ctx, runtime, t, dir, env, "remote", "set-url", "origin", runtime.GitCloneURL); err != nil {
				return "", err
			}
		}
		refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch)
		if _, err := runtimeGitRun(ctx, runtime, t, dir, env, "fetch", "--prune", "origin", refspec); err != nil {
			return "", err
		}
		remoteRef := "origin/" + branch
		if current, _ := runtimeGitRun(ctx, runtime, t, dir, env, "rev-parse", "--abbrev-ref", "HEAD"); current != branch {
			if _, err := runtimeGitRun(ctx, runtime, t, dir, env, "checkout", "-B", branch, remoteRef); err != nil {
				return "", err
			}
		}
		if runtime.GitUpdateStrategy == dto.GithubStrategyReset {
			if _, err := runtimeGitRun(ctx, runtime, t, dir, env, "reset", "--hard", remoteRef); err != nil {
				return "", err
			}
		} else {
			if _, err := runtimeGitRun(ctx, runtime, t, dir, env, "merge", "--ff-only", remoteRef); err != nil {
				return "", err
			}
		}
	}

	head, err := runtimeGitRun(ctx, runtime, t, dir, env, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	if !githubutil.ValidSHA(head) {
		return "", nil
	}
	return head, nil
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
