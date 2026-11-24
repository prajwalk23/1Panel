package service

// fork: deploy scripts of a project. A project has several scripts, exactly one of them is
// the default, and the webhook is linked to exactly one.

import (
	"bufio"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/buserr"
	"github.com/1Panel-dev/1Panel/agent/utils/encrypt"
	githubutil "github.com/1Panel-dev/1Panel/agent/utils/github"
)

const (
	githubDefaultScriptName     = "default"
	githubDefaultScriptTemplate = "# Runs in the project folder after every pull.\n# Variables: $DEPLOY_COMMIT $DEPLOY_BRANCH $DEPLOY_TRIGGER $DEPLOY_PATH $DEPLOY_REPO $DEPLOY_SCRIPT\n"
)

// githubScriptHasCommands is false for scripts that only hold comments and blank lines.
func githubScriptHasCommands(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return true
		}
	}
	return false
}

// githubResolveScript picks the script to run: the requested one, else the one the
// webhook is linked to, else the default. A project without scripts pulls and does
// nothing else, so an empty script is a valid answer.
func githubResolveScript(project *model.GithubProject, scriptID uint) (model.GithubScript, error) {
	if scriptID != 0 {
		script, err := githubRepo.GetScript(repo.WithByID(scriptID), githubRepo.WithByProjectID(project.ID))
		if err != nil {
			return model.GithubScript{}, buserr.New("ErrGithubScriptNotFound")
		}
		return script, nil
	}
	if project.WebhookScriptID != 0 {
		if script, err := githubRepo.GetScript(repo.WithByID(project.WebhookScriptID), githubRepo.WithByProjectID(project.ID)); err == nil {
			return script, nil
		}
	}
	scripts, err := githubRepo.ListScripts(githubRepo.WithByProjectID(project.ID))
	if err != nil {
		return model.GithubScript{}, nil
	}
	for _, item := range scripts {
		if item.IsDefault {
			return item, nil
		}
	}
	if len(scripts) > 0 {
		return scripts[0], nil
	}
	return model.GithubScript{}, nil
}

func githubScriptInfo(script model.GithubScript, linkedID uint) dto.GithubScriptInfo {
	return dto.GithubScriptInfo{
		ID:        script.ID,
		CreatedAt: script.CreatedAt,
		ProjectID: script.ProjectID,
		Name:      script.Name,
		Content:   script.Content,
		RunAsUser: script.RunAsUser,
		IsDefault: script.IsDefault,
		Linked:    script.ID == linkedID,
	}
}

func githubValidScriptName(name string) bool {
	return name != "" && !strings.ContainsAny(name, "\x00\r\n")
}

func githubCheckRunAs(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "root" {
		return "", nil
	}
	if !githubUserPattern.MatchString(name) {
		return "", buserr.New("ErrGithubInvalidUser")
	}
	if _, err := user.Lookup(name); err != nil {
		return "", buserr.New("ErrGithubInvalidUser")
	}
	return name, nil
}

func (g *GithubService) ListScripts(req dto.GithubScriptList) ([]dto.GithubScriptInfo, error) {
	project, err := githubRepo.GetProject(repo.WithByID(req.ProjectID))
	if err != nil {
		return nil, buserr.New("ErrRecordNotFound")
	}
	scripts, err := githubRepo.ListScripts(githubRepo.WithByProjectID(project.ID))
	if err != nil {
		return nil, err
	}
	list := make([]dto.GithubScriptInfo, 0, len(scripts))
	for _, item := range scripts {
		list = append(list, githubScriptInfo(item, project.WebhookScriptID))
	}
	return list, nil
}

func (g *GithubService) CreateScript(req dto.GithubScriptOperate) (*dto.GithubScriptInfo, error) {
	project, err := githubRepo.GetProject(repo.WithByID(req.ProjectID))
	if err != nil {
		return nil, buserr.New("ErrRecordNotFound")
	}
	name := strings.TrimSpace(req.Name)
	if !githubValidScriptName(name) {
		return nil, buserr.New("ErrGithubInvalidName")
	}
	runAs, err := githubCheckRunAs(req.RunAsUser)
	if err != nil {
		return nil, err
	}
	if existing, _ := githubRepo.GetScript(githubRepo.WithByProjectID(project.ID), repo.WithByName(name)); existing.ID != 0 {
		return nil, buserr.New("ErrGithubNameExist")
	}
	scripts, _ := githubRepo.ListScripts(githubRepo.WithByProjectID(project.ID))
	content := strings.ReplaceAll(req.Content, "\r\n", "\n")
	if strings.TrimSpace(content) == "" {
		content = githubDefaultScriptTemplate
	}
	script := model.GithubScript{
		ProjectID: project.ID,
		Name:      name,
		Content:   content,
		RunAsUser: runAs,
		IsDefault: len(scripts) == 0,
	}
	if err := githubRepo.CreateScript(&script); err != nil {
		return nil, err
	}
	info := githubScriptInfo(script, project.WebhookScriptID)
	return &info, nil
}

func (g *GithubService) UpdateScript(req dto.GithubScriptOperate) error {
	script, err := githubRepo.GetScript(repo.WithByID(req.ID), githubRepo.WithByProjectID(req.ProjectID))
	if err != nil {
		return buserr.New("ErrGithubScriptNotFound")
	}
	name := strings.TrimSpace(req.Name)
	if !githubValidScriptName(name) {
		return buserr.New("ErrGithubInvalidName")
	}
	runAs, err := githubCheckRunAs(req.RunAsUser)
	if err != nil {
		return err
	}
	if existing, _ := githubRepo.GetScript(githubRepo.WithByProjectID(script.ProjectID), repo.WithByName(name)); existing.ID != 0 && existing.ID != script.ID {
		return buserr.New("ErrGithubNameExist")
	}
	return githubRepo.UpdateScript(script.ID, map[string]interface{}{
		"name":        name,
		"content":     strings.ReplaceAll(req.Content, "\r\n", "\n"),
		"run_as_user": runAs,
	})
}

func (g *GithubService) DeleteScript(id uint) error {
	script, err := githubRepo.GetScript(repo.WithByID(id))
	if err != nil {
		return buserr.New("ErrGithubScriptNotFound")
	}
	if script.IsDefault {
		return buserr.New("ErrGithubScriptDefault")
	}
	if running := githubQueue.Running(script.ProjectID); running {
		return buserr.New("ErrGithubScriptBusy")
	}
	project, err := githubRepo.GetProject(repo.WithByID(script.ProjectID))
	if err == nil && project.WebhookScriptID == script.ID {
		// The webhook keeps working and only pulls until another script is linked.
		_ = githubRepo.UpdateProject(project.ID, map[string]interface{}{"webhook_script_id": 0})
	}
	return githubRepo.DeleteScript(repo.WithByID(id))
}

// ConfigureWebhook turns the webhook of a project on or off and links it to one script.
// An access key is created when the webhook is switched on the first time.
func (g *GithubService) ConfigureWebhook(req dto.GithubWebhookConfig) (*dto.GithubProjectSecret, error) {
	project, err := githubRepo.GetProject(repo.WithByID(req.ID))
	if err != nil {
		return nil, buserr.New("ErrRecordNotFound")
	}
	result := &dto.GithubProjectSecret{ID: project.ID}
	vars := map[string]interface{}{}
	if !req.Enabled {
		vars["domain_mode"] = dto.GithubDomainNone
		if err := githubRepo.UpdateProject(project.ID, vars); err != nil {
			return nil, err
		}
		return result, nil
	}
	script, err := githubResolveScript(&project, req.ScriptID)
	if err != nil {
		return nil, err
	}
	vars["webhook_script_id"] = script.ID
	if project.WebhookTokenHash == "" || githubWebhookToken(project) == "" {
		vars2, err := githubNewTokenVars()
		if err != nil {
			return nil, err
		}
		for k, v := range vars2 {
			vars[k] = v
		}
	}
	vars["domain_mode"] = dto.GithubDomainDefault
	if err := githubRepo.UpdateProject(project.ID, vars); err != nil {
		return nil, err
	}
	fresh, err := githubRepo.GetProject(repo.WithByID(project.ID))
	if err != nil {
		return nil, err
	}
	info := githubProjectInfo(fresh, loadGithubSetting())
	result.WebhookURL, result.PublicKey = info.WebhookURL, info.PublicKey
	return result, nil
}

// githubNewTokenVars returns the columns of a fresh webhook access key.
func githubNewTokenVars() (map[string]interface{}, error) {
	token, err := githubutil.NewToken()
	if err != nil {
		return nil, err
	}
	encrypted, err := encrypt.StringEncrypt(token)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"webhook_token":      encrypted,
		"webhook_token_hash": githubutil.TokenHash(token),
	}, nil
}

// RefreshWebhookToken replaces the access key. The old URL stops working at once.
func (g *GithubService) RefreshWebhookToken(id uint) (*dto.GithubProjectSecret, error) {
	project, err := githubRepo.GetProject(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New("ErrRecordNotFound")
	}
	vars, err := githubNewTokenVars()
	if err != nil {
		return nil, err
	}
	if err := githubRepo.UpdateProject(id, vars); err != nil {
		return nil, err
	}
	fresh, err := githubRepo.GetProject(repo.WithByID(project.ID))
	if err != nil {
		return nil, err
	}
	info := githubProjectInfo(fresh, loadGithubSetting())
	return &dto.GithubProjectSecret{ID: id, WebhookURL: info.WebhookURL, PublicKey: info.PublicKey}, nil
}

func (g *GithubService) LoadWebhookInfo(id uint) (*dto.GithubWebhookInfo, error) {
	project, err := githubRepo.GetProject(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New("ErrRecordNotFound")
	}
	target := githubWebhookTarget(project)
	info := &dto.GithubWebhookInfo{
		Enabled:     project.DomainMode != dto.GithubDomainNone,
		ScriptID:    project.WebhookScriptID,
		URL:         target.URL,
		Path:        target.Path,
		BindDomain:  target.BindDomain,
		HTTPS:       target.HTTPS,
		ContentType: "application/json",
		Events:      "push",
	}
	if script, err := githubResolveScript(&project, 0); err == nil {
		info.ScriptID, info.ScriptName = script.ID, script.Name
	}
	return info, nil
}

var githubServiceUsers = map[string]bool{"www": true, "www-data": true, "nginx": true, "apache": true}

// ListSystemUsers returns the accounts a script can run as: normal users and common
// web server accounts. Root is the default and is not listed.
func (g *GithubService) ListSystemUsers() ([]string, error) {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return []string{}, nil
	}
	defer file.Close()
	users := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) < 7 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		name := fields[0]
		uid, convErr := strconv.Atoi(fields[2])
		if convErr != nil || uid == 0 || !githubUserPattern.MatchString(name) {
			continue
		}
		if (uid >= 1000 && uid < 65534) || githubServiceUsers[name] {
			users = append(users, name)
		}
	}
	sort.Strings(users)
	return users, nil
}
