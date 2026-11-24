package service

// fork: webhook of the GitHub section. Requests arrive on the panel port (/hook?access_key=...),
// the panel core forwards them over the agent socket to HandleGithubHook. The access key in the
// URL is the only secret. This file also holds startup work and the server restart re-run.

import (
	"strconv"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/i18n"
	githubutil "github.com/1Panel-dev/1Panel/agent/utils/github"
	"github.com/shirou/gopsutil/v4/host"
)

const (
	githubSettingLastBoot = "GithubLastBootTime"
	githubBootDelay       = 30 * time.Second
)

// GithubHookRequest is one webhook delivery as the panel core hands it over.
type GithubHookRequest struct {
	Token    string
	Event    string // X-GitHub-Event, empty for other providers
	Delivery string
	ClientIP string
	Body     []byte
}

// GithubHookResult is what the sender gets back.
type GithubHookResult struct {
	Code    int
	Message string
	Reason  string
}

func githubHookResult(code int, message, reason string) GithubHookResult {
	return GithubHookResult{Code: code, Message: message, Reason: reason}
}

// HandleGithubHook checks a delivery and queues a deploy when it should run.
func HandleGithubHook(req GithubHookRequest) GithubHookResult {
	token := strings.TrimSpace(req.Token)
	if token == "" || len(token) > 256 {
		return githubHookResult(404, "unknown webhook", "")
	}
	project, err := githubRepo.GetProject(githubRepo.WithByWebhookTokenHash(githubutil.TokenHash(token)))
	if err != nil || !githubutil.TokenEqual(githubWebhookToken(project), token) {
		global.LOG.Warnf("[github] rejected delivery from %s: unknown access key", req.ClientIP)
		return githubHookResult(404, "unknown webhook", "")
	}
	if !githubLimiter.Allow(project.ID) {
		return githubHookResult(429, "rate limited", "")
	}
	if project.DomainMode == dto.GithubDomainNone {
		return githubHookResult(403, "webhook is turned off for this project", "")
	}
	if !project.Enabled {
		return githubHookResult(403, "project disabled", "")
	}

	delivery := req.Delivery
	if !githubDeliveryPattern.MatchString(delivery) {
		delivery = ""
	}
	event := strings.TrimSpace(req.Event)
	if event == githubutil.EventPing {
		return githubHookResult(200, "pong", "")
	}
	fromGithub := event != ""
	if fromGithub && event != githubutil.EventPush {
		return githubHookResult(202, "ignored", "event "+githubTrim(event, 64))
	}

	push, parseErr := githubutil.ParsePush(req.Body)
	if fromGithub {
		if parseErr != nil {
			return githubHookResult(400, "invalid payload", "")
		}
		switch result := githubutil.Match(project.Repo, project.Branch, event, push); result {
		case githubutil.MatchDeploy:
		case githubutil.MatchWrongRepo:
			return githubHookResult(400, "repository does not match project", "")
		default:
			return githubHookResult(202, "ignored", result.String())
		}
	} else if parseErr == nil {
		// Other git hosts (GitLab, Gitee, Gitea): a push of another branch is ignored.
		if push.Ref != "" && push.Ref != "refs/heads/"+project.Branch {
			return githubHookResult(202, "ignored", githubutil.MatchIgnoreBranch.String())
		}
		if push.Deleted {
			return githubHookResult(202, "ignored", githubutil.MatchIgnoreDeleted.String())
		}
	}

	commit := ""
	if githubutil.ValidSHA(push.After) {
		commit = push.After
	}
	ref := githubTrim(push.Ref, 255)
	if ref == "" {
		ref = "refs/heads/" + project.Branch
	}
	deploy := githubDeployReq{
		Trigger:    dto.GithubTriggerWebhook,
		Ref:        ref,
		Commit:     commit,
		Pusher:     githubTrim(push.Pusher.Name, 100),
		DeliveryID: delivery,
	}

	if ignore := githubutil.ParseIgnorePaths(project.IgnorePaths); githubutil.MatchIgnored(ignore, push.ChangedFiles()) {
		now := time.Now()
		_ = githubRepo.CreateLog(&model.GithubDeployLog{
			ProjectID:  project.ID,
			Trigger:    deploy.Trigger,
			DeliveryID: deploy.DeliveryID,
			Ref:        deploy.Ref,
			Commit:     deploy.Commit,
			Pusher:     deploy.Pusher,
			Status:     dto.GithubStatusSkipped,
			Message:    i18n.GetMsgByKey("GithubIgnoredPaths"),
			StartedAt:  now,
			FinishedAt: now,
		})
		return githubHookResult(202, "ignored", "ignored paths")
	}

	state, err := githubSubmit(project, deploy)
	if err != nil {
		return githubHookResult(500, "queue failed", "")
	}
	global.LOG.Infof("[github] delivery %s for project %d from %s: %s", delivery, project.ID, req.ClientIP, state)
	return githubHookResult(202, state, "")
}

// InitGithub is run once from business.Init().
func InitGithub() {
	defer func() {
		if r := recover(); r != nil {
			global.LOG.Errorf("[github] init panic: %v", r)
		}
	}()
	if err := githubutil.EnforceKeyPerms(githubKeyRoot()); err != nil {
		global.LOG.Errorf("[github] enforce key permissions failed: %v", err)
	}
	githubFailInterrupted()
	githubDropOrphanKeys()
	go githubBootRun()
}

// githubBootRun runs the webhook script of every project whose webhook is on, once, after the
// server (not just the panel) restarted. It does not pull: it brings services that the deploy
// script starts back up.
func githubBootRun() {
	defer func() {
		if r := recover(); r != nil {
			global.LOG.Errorf("[github] boot run panic: %v", r)
		}
	}()
	boot, err := host.BootTime()
	if err != nil || boot == 0 {
		return
	}
	lastText, _ := settingRepo.GetValueByKey(githubSettingLastBoot)
	last, _ := strconv.ParseUint(strings.TrimSpace(lastText), 10, 64)
	_ = settingRepo.UpdateOrCreate(githubSettingLastBoot, strconv.FormatUint(boot, 10))
	// 0: first start after the upgrade, nothing to compare with. The slack absorbs boot time jitter.
	if last == 0 || boot <= last+60 {
		return
	}
	time.Sleep(githubBootDelay)
	projects, err := githubRepo.ListProjects(githubRepo.WithByEnabled(true))
	if err != nil {
		return
	}
	for i := range projects {
		project := projects[i]
		if project.DomainMode == dto.GithubDomainNone {
			continue
		}
		script, err := githubResolveScript(&project, 0)
		if err != nil || !githubScriptHasCommands(script.Content) {
			continue
		}
		if _, err := githubSubmit(project, githubDeployReq{
			Trigger:    dto.GithubTriggerBoot,
			Ref:        "refs/heads/" + project.Branch,
			ScriptOnly: true,
		}); err != nil {
			global.LOG.Errorf("[github] boot run of project %d failed: %v", project.ID, err)
		}
	}
}
