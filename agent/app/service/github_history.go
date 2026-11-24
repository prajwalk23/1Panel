package service

// fork: git environment of a project, commit history, deploy summary and rollback versions.

import (
	"context"
	"os"
	"path"
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

// githubProjectEnv returns the environment git runs with for a project: the SSH key or the
// access token of the project, whichever it uses.
func githubProjectEnv(project model.GithubProject, setting dto.GithubSetting) ([]string, error) {
	env := []string{"GIT_TERMINAL_PROMPT=0"}
	if project.KeyMode == dto.GithubKeyModeToken {
		token, err := encrypt.StringDecrypt(project.Token)
		if err != nil || token == "" {
			return nil, buserr.New("ErrGithubTokenRequired")
		}
		return append(env, githubutil.TokenEnv(project.TokenUser, token)...), nil
	}
	key, err := githubProjectKey(project, setting)
	if err != nil {
		return nil, err
	}
	if key.ID != 0 {
		env = append(env, "GIT_SSH_COMMAND="+githubutil.SSHCommand(key.KeyPath, githubKnownHosts()))
	}
	return env, nil
}

// githubRollbackVersions returns the commits of the last successful deploys, as many as the
// project keeps (Deployment records). Server restart re-runs are not versions.
func githubRollbackVersions(project model.GithubProject) map[string]bool {
	keep := project.KeepRecords
	if keep <= 0 {
		keep = githubDefaultKeepRecords
	}
	logs, _ := githubRepo.ListLogs(githubRepo.WithByProjectID(project.ID), githubRepo.WithByLogStatus(dto.GithubStatusSuccess), repo.WithOrderDesc("created_at"))
	versions := map[string]bool{}
	for _, item := range logs {
		if item.Commit == "" || item.Trigger == dto.GithubTriggerBoot || versions[item.Commit] {
			continue
		}
		if len(versions) >= keep {
			break
		}
		versions[item.Commit] = true
	}
	return versions
}

// githubRestartLinkedRuntime restarts the runtime of a runtime project after a deploy.
func githubRestartLinkedRuntime(project model.GithubProject) {
	if project.RuntimeID == 0 {
		return
	}
	runtime, err := runtimeRepo.GetFirst(context.Background(), repo.WithByID(project.RuntimeID))
	if err != nil {
		return
	}
	if err := restartRuntime(runtime); err != nil {
		global.LOG.Errorf("[github] restart runtime %s after deploy failed: %v", runtime.Name, err)
	}
}

func (g *GithubService) LoadSummary(id uint) (*dto.GithubProjectSummary, error) {
	project, err := githubRepo.GetProject(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New("ErrRecordNotFound")
	}
	summary := &dto.GithubProjectSummary{Project: githubProjectInfo(project, loadGithubSetting())}
	logs, _ := githubRepo.ListLogs(githubRepo.WithByProjectID(id), githubRepo.WithByLogStatus(dto.GithubStatusSuccess), repo.WithOrderDesc("created_at"))
	for _, item := range logs {
		if item.Trigger == dto.GithubTriggerBoot {
			continue
		}
		summary.Commit = item.Commit
		summary.CommitMessage = item.CommitMessage
		summary.CommitAuthor = item.CommitAuthor
		summary.DeployedAt = item.FinishedAt
		summary.Duration = item.Duration
		break
	}
	if summary.Commit == "" {
		summary.Commit = project.LastCommit
	}
	return summary, nil
}

// SearchCommits lists the commits of a branch of the project's repository.
func (g *GithubService) SearchCommits(req dto.GithubCommitSearch) (int64, []dto.GithubCommitInfo, error) {
	project, err := githubRepo.GetProject(repo.WithByID(req.ID))
	if err != nil {
		return 0, nil, buserr.New("ErrRecordNotFound")
	}
	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = project.Branch
	}
	if !githubutil.ValidBranch(branch) {
		return 0, nil, buserr.New("ErrGithubInvalidBranch")
	}
	if !cmd.Which("git") {
		return 0, nil, buserr.New("ErrGithubGitNotFound")
	}
	dir := project.ProjectPath
	if _, err := os.Stat(path.Join(dir, ".git")); err != nil {
		return 0, []dto.GithubCommitInfo{}, nil
	}
	env, err := githubProjectEnv(project, loadGithubSetting())
	if err != nil {
		return 0, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	remoteRef := "origin/" + branch
	refspec := "+refs/heads/" + branch + ":refs/remotes/" + remoteRef
	if _, err := githubGitQuiet(ctx, dir, env, "fetch", "--prune", "origin", refspec); err != nil {
		return 0, nil, buserr.WithErr("ErrGithubFetchFailed", err)
	}
	countText, err := githubGitQuiet(ctx, dir, env, "rev-list", "--count", remoteRef)
	if err != nil {
		return 0, nil, err
	}
	total, _ := strconv.ParseInt(strings.TrimSpace(countText), 10, 64)

	size := req.PageSize
	if size <= 0 || size > 100 {
		size = 10
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	out, err := githubGitQuiet(ctx, dir, env, "log", remoteRef,
		"--skip="+strconv.Itoa((page-1)*size), "--max-count="+strconv.Itoa(size),
		"--format=%H%x1f%an%x1f%at%x1f%s")
	if err != nil {
		return 0, nil, err
	}
	list := make([]dto.GithubCommitInfo, 0, size)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "\x1f", 4)
		if len(parts) != 4 || !githubutil.ValidSHA(parts[0]) {
			continue
		}
		seconds, _ := strconv.ParseInt(parts[2], 10, 64)
		list = append(list, dto.GithubCommitInfo{
			Commit:  parts[0],
			Author:  githubTrim(parts[1], 100),
			Time:    time.Unix(seconds, 0),
			Message: githubTrim(parts[3], 500),
			Active:  parts[0] == project.LastCommit,
		})
	}
	return total, list, nil
}
