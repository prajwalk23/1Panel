package service

// fork: GitHub section - webhook listener, deploy queue and runner, deploy logs.

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/app/task"
	"github.com/1Panel-dev/1Panel/agent/buserr"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/i18n"
	"github.com/1Panel-dev/1Panel/agent/utils/cmd"
	githubutil "github.com/1Panel-dev/1Panel/agent/utils/github"
)

const (
	githubTaskScope      = "Github"
	githubMaxBody        = 10 << 20
	githubMessageMaxSize = 4000
)

var (
	githubQueue   = githubutil.NewQueue(runGithubDeploy)
	githubLimiter = githubutil.NewLimiter(2*time.Second, 20)

	githubDeliveryPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
)

type githubDeployReq struct {
	LogID      uint
	Trigger    string
	Ref        string
	Commit     string
	Pusher     string
	DeliveryID string
	ScriptID   uint // 0: the script the webhook is linked to

	TargetCommit string // check this commit out instead of following the branch (deploy a commit, rollback)
	ScriptOnly   bool   // run the script only, no pull (server restart)
}

// ---------- queue ----------

func (g *GithubService) Deploy(req dto.GithubProjectDeploy) error {
	project, err := githubRepo.GetProject(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New("ErrRecordNotFound")
	}
	if _, err := githubResolveScript(&project, req.ScriptID); err != nil {
		return err
	}
	deploy := githubDeployReq{Trigger: dto.GithubTriggerManual, Ref: "refs/heads/" + project.Branch, ScriptID: req.ScriptID}
	if commit := strings.TrimSpace(req.Commit); commit != "" {
		if !githubutil.ValidSHA(commit) {
			return buserr.New("ErrGithubInvalidCommit")
		}
		deploy.TargetCommit, deploy.Commit = commit, commit
	}
	if req.Rollback {
		if deploy.TargetCommit == "" {
			return buserr.New("ErrGithubInvalidCommit")
		}
		deploy.Trigger = dto.GithubTriggerRollback
	}
	_, err = githubSubmit(project, deploy)
	return err
}

func (g *GithubService) Cancel(id uint) error {
	ok, dropped := githubQueue.Cancel(id)
	githubMarkDropped(dropped, "GithubCanceled")
	if !ok {
		return buserr.New("ErrGithubNotRunning")
	}
	return nil
}

func githubSubmit(project model.GithubProject, req githubDeployReq) (string, error) {
	logItem := model.GithubDeployLog{
		ProjectID:  project.ID,
		Trigger:    req.Trigger,
		DeliveryID: req.DeliveryID,
		Ref:        req.Ref,
		Commit:     req.Commit,
		Pusher:     req.Pusher,
		Status:     dto.GithubStatusWaiting,
	}
	if err := githubRepo.CreateLog(&logItem); err != nil {
		return "", err
	}
	req.LogID = logItem.ID
	result, dropped := githubQueue.Submit(project.ID, req)
	githubMarkDropped(dropped, "GithubSuperseded")
	switch result {
	case githubutil.Started:
		return "started", nil
	case githubutil.Replaced:
		return "queued (replaced older)", nil
	default:
		return "queued", nil
	}
}

func githubMarkDropped(dropped *githubDeployReq, msgKey string) {
	if dropped == nil || dropped.LogID == 0 {
		return
	}
	status := dto.GithubStatusSkipped
	if msgKey == "GithubCanceled" {
		status = dto.GithubStatusCanceled
	}
	_ = githubRepo.UpdateLog(dropped.LogID, map[string]interface{}{
		"status":      status,
		"message":     i18n.GetMsgByKey(msgKey),
		"finished_at": time.Now(),
	})
}

// githubFailInterrupted marks runs left over from a previous agent process.
func githubFailInterrupted() {
	logs, err := githubRepo.ListLogs(githubRepo.WithByLogStatus(dto.GithubStatusWaiting, dto.GithubStatusRunning))
	if err != nil {
		return
	}
	for _, item := range logs {
		_ = githubRepo.UpdateLog(item.ID, map[string]interface{}{
			"status":      dto.GithubStatusFailed,
			"message":     i18n.GetMsgByKey("GithubInterrupted"),
			"finished_at": time.Now(),
		})
		_ = githubRepo.UpdateProject(item.ProjectID, map[string]interface{}{"last_status": dto.GithubStatusFailed})
	}
}

// ---------- runner ----------

func runGithubDeploy(ctx context.Context, projectID uint, req githubDeployReq) {
	finish := func(status, message, commit string) {
		vars := map[string]interface{}{"status": status, "message": githubTrim(message, githubMessageMaxSize), "finished_at": time.Now()}
		if commit != "" {
			vars["commit"] = commit
		}
		_ = githubRepo.UpdateLog(req.LogID, vars)
		projectVars := map[string]interface{}{"last_status": status}
		if commit != "" {
			projectVars["last_commit"] = commit
		}
		_ = githubRepo.UpdateProject(projectID, projectVars)
	}

	project, err := githubRepo.GetProject(repo.WithByID(projectID))
	if err != nil {
		finish(dto.GithubStatusFailed, i18n.GetMsgByKey("ErrRecordNotFound"), "")
		return
	}
	setting := loadGithubSetting()
	deployTask, err := task.NewTaskWithOps(project.Name, task.TaskPull, githubTaskScope, "", project.ID)
	if err != nil {
		finish(dto.GithubStatusFailed, err.Error(), "")
		return
	}
	now := time.Now()
	_ = githubRepo.UpdateLog(req.LogID, map[string]interface{}{"status": dto.GithubStatusRunning, "task_id": deployTask.TaskID, "started_at": now})
	_ = githubRepo.UpdateProject(projectID, map[string]interface{}{"last_status": dto.GithubStatusRunning, "last_run_at": now})

	timeout := time.Duration(project.Timeout) * time.Second
	if timeout <= 0 {
		timeout = githubDefaultTimeout * time.Second
	}
	env, err := githubProjectEnv(project, setting)
	if err != nil {
		githubCloseTask(deployTask, err)
		finish(dto.GithubStatusFailed, err.Error(), "")
		return
	}

	headCommit := ""
	if req.ScriptOnly {
		headCommit = project.LastCommit
	} else {
		deployTask.AddSubTaskWithAliasAndOps("GithubStepSync", func(t *task.Task) error {
			commit, err := githubSyncRepo(ctx, t, project, env, timeout, req.TargetCommit)
			headCommit = commit
			return err
		}, nil, 0, timeout+time.Minute)
	}
	script, scriptErr := githubResolveScript(&project, req.ScriptID)
	if scriptErr != nil {
		githubCloseTask(deployTask, scriptErr)
		finish(dto.GithubStatusFailed, scriptErr.Error(), "")
		return
	}
	if githubScriptHasCommands(script.Content) {
		deployTask.AddSubTaskWithAliasAndOps("GithubStepDeploy", func(t *task.Task) error {
			return githubRunScript(ctx, t, project, script, setting, req, env, headCommit, timeout)
		}, nil, 0, timeout+time.Minute)
	}

	runErr := deployTask.Execute()
	logVars := map[string]interface{}{"duration": time.Since(now).Milliseconds()}
	if !req.ScriptOnly && headCommit != "" {
		if info, infoErr := githubGitQuiet(ctx, project.ProjectPath, env, "log", "-1", "--format=%an%x1f%s", headCommit); infoErr == nil {
			if parts := strings.SplitN(info, "\x1f", 2); len(parts) == 2 {
				logVars["commit_author"] = githubTrim(parts[0], 100)
				logVars["commit_message"] = githubTrim(parts[1], 500)
			}
		}
	}
	_ = githubRepo.UpdateLog(req.LogID, logVars)
	switch {
	case ctx.Err() != nil:
		finish(dto.GithubStatusCanceled, i18n.GetMsgByKey("GithubCanceled"), headCommit)
	case runErr != nil:
		finish(dto.GithubStatusFailed, runErr.Error(), headCommit)
	default:
		finish(dto.GithubStatusSuccess, "", headCommit)
		if !req.ScriptOnly {
			githubRestartLinkedRuntime(project)
		}
	}
	githubPruneLogs(projectID, setting.LogRetention)
}

// githubCloseTask records a task that failed before any sub task ran.
func githubCloseTask(t *task.Task, err error) {
	t.AddSubTask(i18n.GetMsgByKey("GithubStepSync"), func(*task.Task) error { return err }, nil)
	_ = t.Execute()
}

func githubGit(ctx context.Context, t *task.Task, dir string, env []string, timeout time.Duration, args ...string) (string, error) {
	full := append([]string{"-c", "safe.directory=*"}, args...)
	mgr := cmd.NewCommandMgr(cmd.WithTask(*t), cmd.WithContext(ctx), cmd.WithTimeout(timeout), cmd.WithWorkDir(dir), cmd.WithEnv(env...))
	out, err := mgr.RunWithStdout("git", full...)
	return strings.TrimSpace(out), err
}

func githubGitQuiet(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	full := append([]string{"-c", "safe.directory=*"}, args...)
	mgr := cmd.NewCommandMgr(cmd.WithContext(ctx), cmd.WithTimeout(time.Minute), cmd.WithWorkDir(dir), cmd.WithEnv(env...))
	out, err := mgr.RunWithStdout("git", full...)
	return strings.TrimSpace(out), err
}

// githubSyncRepo clones or updates the project folder and returns HEAD.
// With target set the checkout ends on that commit instead of following the branch.
func githubSyncRepo(ctx context.Context, t *task.Task, project model.GithubProject, env []string, timeout time.Duration, target string) (string, error) {
	if !cmd.Which("git") {
		return "", buserr.New("ErrGithubGitNotFound")
	}
	dir := project.ProjectPath
	branch := project.Branch
	if _, err := os.Stat(path.Join(dir, ".git")); err != nil {
		entries, statErr := os.ReadDir(dir)
		if statErr == nil && len(entries) != 0 {
			return "", buserr.WithName("ErrGithubPathNotEmpty", dir)
		}
		parent := path.Dir(dir)
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return "", err
		}
		t.Logf("git clone --branch %s --single-branch %s %s", branch, project.CloneURL, dir)
		if _, err := githubGit(ctx, t, parent, env, timeout, "clone", "--branch", branch, "--single-branch", "--", project.CloneURL, dir); err != nil {
			return "", err
		}
	} else {
		if current, _ := githubGitQuiet(ctx, dir, env, "remote", "get-url", "origin"); current != project.CloneURL {
			t.Logf("git remote set-url origin %s", project.CloneURL)
			if _, err := githubGit(ctx, t, dir, env, timeout, "remote", "set-url", "origin", project.CloneURL); err != nil {
				return "", err
			}
		}
		t.Logf("git fetch origin %s", branch)
		refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch)
		if _, err := githubGit(ctx, t, dir, env, timeout, "fetch", "--prune", "origin", refspec); err != nil {
			return "", err
		}
		remoteRef := "origin/" + branch
		if current, _ := githubGitQuiet(ctx, dir, env, "rev-parse", "--abbrev-ref", "HEAD"); current != branch {
			t.Logf("git checkout -B %s %s", branch, remoteRef)
			if _, err := githubGit(ctx, t, dir, env, timeout, "checkout", "-B", branch, remoteRef); err != nil {
				return "", err
			}
		}
		if target != "" {
			// checked out below
		} else if project.UpdateStrategy == dto.GithubStrategyReset {
			t.Logf("git reset --hard %s", remoteRef)
			if _, err := githubGit(ctx, t, dir, env, timeout, "reset", "--hard", remoteRef); err != nil {
				return "", err
			}
		} else {
			t.Logf("git merge --ff-only %s", remoteRef)
			if _, err := githubGit(ctx, t, dir, env, timeout, "merge", "--ff-only", remoteRef); err != nil {
				return "", err
			}
		}
	}
	if target != "" {
		if _, err := githubGitQuiet(ctx, dir, env, "cat-file", "-e", target+"^{commit}"); err != nil {
			return "", buserr.New("ErrGithubCommitNotFound")
		}
		t.Logf("git reset --hard %s", target)
		if _, err := githubGit(ctx, t, dir, env, timeout, "reset", "--hard", target); err != nil {
			return "", err
		}
	}
	head, err := githubGitQuiet(ctx, dir, env, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	t.Logf("HEAD %s", head)
	if !githubutil.ValidSHA(head) {
		return "", nil
	}
	return head, nil
}

func githubChownTree(ctx context.Context, t *task.Task, userName, dir string) error {
	u, err := user.Lookup(userName)
	if err != nil {
		return buserr.New("ErrGithubInvalidUser")
	}
	owner := u.Uid + ":" + u.Gid
	t.Logf("chown -R %s %s", userName, dir)
	return cmd.NewCommandMgr(cmd.WithTask(*t), cmd.WithContext(ctx), cmd.WithTimeout(5*time.Minute)).Run("chown", "-R", owner, dir)
}

func githubRunScript(ctx context.Context, t *task.Task, project model.GithubProject, script model.GithubScript, setting dto.GithubSetting, req githubDeployReq, gitEnv []string, headCommit string, timeout time.Duration) error {
	shell := setting.DefaultShell
	if !cmd.Which(shell) {
		shell = "sh"
	}
	commit := headCommit
	if commit == "" && githubutil.ValidSHA(req.Commit) {
		commit = req.Commit
	}
	env := []string{
		"DEPLOY_PROJECT=" + project.Name,
		"DEPLOY_REPO=" + project.Repo,
		"DEPLOY_BRANCH=" + project.Branch,
		"DEPLOY_COMMIT=" + commit,
		"DEPLOY_TRIGGER=" + req.Trigger,
		"DEPLOY_PATH=" + project.ProjectPath,
		"DEPLOY_SCRIPT=" + script.Name,
	}
	name, args := shell, []string{"-c", script.Content}
	if script.RunAsUser != "" {
		if !cmd.Which("runuser") {
			return buserr.New("ErrGithubRunuserNotFound")
		}
		if err := githubChownTree(ctx, t, script.RunAsUser, project.ProjectPath); err != nil {
			return err
		}
		name, args = "runuser", append([]string{"-u", script.RunAsUser, "--", shell}, args...)
	} else {
		// Root scripts get the git credentials of the project, so `git pull` works on private repositories.
		env = append(env, gitEnv...)
	}
	t.Logf("%s: %s, %s (%s)", i18n.GetMsgByKey("GithubStepDeploy"), script.Name, shell, project.ProjectPath)
	mgr := cmd.NewCommandMgr(cmd.WithTask(*t), cmd.WithContext(ctx), cmd.WithTimeout(timeout), cmd.WithWorkDir(project.ProjectPath), cmd.WithEnv(env...))
	return mgr.Run(name, args...)
}

// ---------- logs ----------

func (g *GithubService) SearchLogs(req dto.GithubLogSearch) (int64, []dto.GithubLogInfo, error) {
	var opts []repo.DBOption
	if req.ProjectID != 0 {
		opts = append(opts, githubRepo.WithByProjectID(req.ProjectID))
	}
	if req.Status != "" {
		opts = append(opts, githubRepo.WithByLogStatus(req.Status))
	}
	if req.Trigger != "" {
		opts = append(opts, githubRepo.WithByTrigger(req.Trigger))
	}
	total, logs, err := githubRepo.PageLogs(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	names := map[uint]string{}
	projects, _ := githubRepo.ListProjects()
	for _, project := range projects {
		names[project.ID] = project.Name
	}
	var current model.GithubProject
	versions := map[string]bool{}
	if req.ProjectID != 0 {
		current, _ = githubRepo.GetProject(repo.WithByID(req.ProjectID))
		versions = githubRollbackVersions(current)
	}
	list := make([]dto.GithubLogInfo, 0, len(logs))
	for _, item := range logs {
		info := dto.GithubLogInfo{
			ID:          item.ID,
			CreatedAt:   item.CreatedAt,
			ProjectID:   item.ProjectID,
			ProjectName: names[item.ProjectID],
			Trigger:     item.Trigger,
			DeliveryID:  item.DeliveryID,
			Ref:         item.Ref,
			Commit:      item.Commit,
			Pusher:      item.Pusher,
			Status:      item.Status,
			TaskID:      item.TaskID,
			Message:     item.Message,
			StartedAt:   item.StartedAt,
			FinishedAt:  item.FinishedAt,

			CommitMessage: item.CommitMessage,
			CommitAuthor:  item.CommitAuthor,
			Duration:      item.Duration,
		}
		if current.ID != 0 && item.Status == dto.GithubStatusSuccess && item.Commit != "" {
			info.Active = item.Commit == current.LastCommit
			info.Rollbackable = !info.Active && versions[item.Commit]
		}
		list = append(list, info)
	}
	return total, list, nil
}

func (g *GithubService) CleanLogs(req dto.GithubLogClean) error {
	opts := []repo.DBOption{githubRepo.WithByLogStatus(dto.GithubStatusSuccess, dto.GithubStatusFailed, dto.GithubStatusCanceled, dto.GithubStatusSkipped)}
	if req.ProjectID != 0 {
		opts = append(opts, githubRepo.WithByProjectID(req.ProjectID))
	}
	if req.Trigger != "" {
		opts = append(opts, githubRepo.WithByTrigger(req.Trigger))
	}
	logs, err := githubRepo.ListLogs(opts...)
	if err != nil {
		return err
	}
	githubDeleteLogs(logs)
	return nil
}

func githubPruneLogs(projectID uint, keep int) {
	if keep <= 0 {
		return
	}
	logs, err := githubRepo.ListLogs(githubRepo.WithByProjectID(projectID), repo.WithOrderDesc("created_at"))
	if err != nil || len(logs) <= keep {
		return
	}
	var old []model.GithubDeployLog
	for _, item := range logs[keep:] {
		if item.Status == dto.GithubStatusRunning || item.Status == dto.GithubStatusWaiting {
			continue
		}
		old = append(old, item)
	}
	githubDeleteLogs(old)
}

func githubDeleteLogs(logs []model.GithubDeployLog) {
	for _, item := range logs {
		if item.TaskID != "" {
			_ = os.Remove(path.Join(global.Dir.TaskDir, githubTaskScope, item.TaskID+".log"))
			_ = taskRepo.Delete(taskRepo.WithByID(item.TaskID))
		}
		_ = githubRepo.DeleteLog(repo.WithByID(item.ID))
	}
}

func githubTrim(s string, max int) string {
	if len(s) > max {
		s = s[:max]
	}
	return strings.ToValidUTF8(s, "")
}
