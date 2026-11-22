package service

// fork: repository check and project key helpers of the GitHub section.

import (
	"context"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/buserr"
	"github.com/1Panel-dev/1Panel/agent/utils/cmd"
	githubutil "github.com/1Panel-dev/1Panel/agent/utils/github"
)

const githubMaxBranches = 500

// CheckRepo runs "git ls-remote" with the chosen key. A repository that cannot be
// reached is a normal answer (OK=false with git's message), not an API error.
func (g *GithubService) CheckRepo(req dto.GithubRepoCheck) (*dto.GithubRepoCheckInfo, error) {
	cloneURL := strings.TrimSpace(req.CloneURL)
	if !githubValidCloneURL(cloneURL) {
		return nil, buserr.New("ErrGithubInvalidCloneURL")
	}
	if !cmd.Which("git") {
		return nil, buserr.New("ErrGithubGitNotFound")
	}
	env := []string{"GIT_TERMINAL_PROMPT=0"}
	keyPath := ""
	switch req.KeyMode {
	case dto.GithubKeyModeNone:
		if githubutil.IsSSHCloneURL(cloneURL) {
			return nil, buserr.New("ErrGithubKeyRequired")
		}
	case dto.GithubKeyModeShared:
		setting := loadGithubSetting()
		if setting.SharedKeyID == 0 {
			return nil, buserr.New("ErrGithubSharedKeyNotSet")
		}
		key, err := githubRepo.GetKey(repo.WithByID(setting.SharedKeyID))
		if err != nil {
			return nil, buserr.New("ErrGithubKeyNotFound")
		}
		keyPath = key.KeyPath
	case dto.GithubKeyModeProject, dto.GithubKeyModeCustom:
		key, err := githubRepo.GetKey(repo.WithByID(req.KeyID))
		if err != nil || (req.KeyMode == dto.GithubKeyModeProject && key.Type != dto.GithubKeyTypeProject) {
			return nil, buserr.New("ErrGithubKeyNotFound")
		}
		keyPath = key.KeyPath
	default:
		return nil, buserr.New("ErrGithubInvalidKeyMode")
	}
	if keyPath != "" {
		env = append(env, "GIT_SSH_COMMAND="+githubutil.SSHCommand(keyPath, githubKnownHosts()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := githubGitQuiet(ctx, os.TempDir(), env, "ls-remote", "--symref", "--", cloneURL, "HEAD", "refs/heads/*")
	if err != nil {
		return &dto.GithubRepoCheckInfo{Message: githubTrim(strings.TrimSpace(err.Error()), 600)}, nil
	}
	info := &dto.GithubRepoCheckInfo{OK: true, Branches: []string{}}
	seen := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "ref: refs/heads/"):
			if fields := strings.Fields(strings.TrimPrefix(line, "ref: refs/heads/")); len(fields) > 0 {
				info.DefaultBranch = fields[0]
			}
		default:
			fields := strings.Fields(line)
			if len(fields) != 2 || !strings.HasPrefix(fields[1], "refs/heads/") {
				continue
			}
			name := strings.TrimPrefix(fields[1], "refs/heads/")
			if githubutil.ValidBranch(name) && !seen[name] {
				seen[name] = true
				info.Branches = append(info.Branches, name)
			}
		}
	}
	sort.Strings(info.Branches)
	if len(info.Branches) > githubMaxBranches {
		info.Branches = info.Branches[:githubMaxBranches]
	}
	if info.DefaultBranch == "" && len(info.Branches) > 0 {
		info.DefaultBranch = info.Branches[0]
	}
	return info, nil
}

// githubPickProjectKey returns the project key a project uses. current is the key
// the project already has in project mode (kept as it is); wantID is a key the form
// created beforehand so its public key could be added to GitHub before saving.
// created is true when a new key was made here and must be dropped if saving fails.
func githubPickProjectKey(name string, wantID, current uint) (keyID uint, created bool, err error) {
	if current != 0 {
		return current, false, nil
	}
	if wantID != 0 {
		key, getErr := githubRepo.GetKey(repo.WithByID(wantID))
		if getErr != nil || key.Type != dto.GithubKeyTypeProject {
			return 0, false, buserr.New("ErrGithubKeyNotFound")
		}
		projects, _ := githubRepo.ListProjects()
		for _, project := range projects {
			if project.KeyMode == dto.GithubKeyModeProject && project.KeyID == key.ID {
				return 0, false, buserr.WithName("ErrGithubKeyInUse", project.Name)
			}
		}
		_ = githubRepo.UpdateKey(key.ID, map[string]interface{}{"name": name})
		return key.ID, false, nil
	}
	key, createErr := githubCreateKeyRecord(name, dto.GithubKeyTypeProject, nil)
	if createErr != nil {
		return 0, false, createErr
	}
	return key.ID, true, nil
}

// githubDropOrphanKeys removes project keys that no project uses, such as ones
// made by a form that was closed without saving. Recent keys are kept.
func githubDropOrphanKeys() {
	keys, err := githubRepo.ListKeys()
	if err != nil {
		return
	}
	projects, _ := githubRepo.ListProjects()
	used := map[uint]bool{}
	for _, project := range projects {
		if project.KeyID != 0 {
			used[project.KeyID] = true
		}
	}
	for _, key := range keys {
		if key.Type == dto.GithubKeyTypeProject && !used[key.ID] && time.Since(key.CreatedAt) > 24*time.Hour {
			githubDropKey(key.ID)
		}
	}
}
