package service

// fork: SSH deploy keys of the GitHub section. Private keys stay on disk
// (0600 under <DataDir>/github/keys/<id>/) and are never returned by the API.

import (
	"errors"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/buserr"
	"github.com/1Panel-dev/1Panel/agent/global"
	githubutil "github.com/1Panel-dev/1Panel/agent/utils/github"
)

func githubBaseDir() string {
	return path.Join(global.Dir.DataDir, "github")
}

func githubKeyRoot() string {
	return path.Join(githubBaseDir(), "keys")
}

func githubKnownHosts() string {
	return path.Join(githubBaseDir(), "known_hosts")
}

func (g *GithubService) SearchKeys(req dto.GithubKeySearch) (int64, []dto.GithubKeyInfo, error) {
	total, keys, err := githubRepo.PageKeys(req.Page, req.PageSize, repo.WithByLikeName(req.Info))
	if err != nil {
		return 0, nil, err
	}
	setting := loadGithubSetting()
	projects, _ := githubRepo.ListProjects()
	list := make([]dto.GithubKeyInfo, 0, len(keys))
	for _, key := range keys {
		list = append(list, githubKeyInfo(key, setting, projects))
	}
	return total, list, nil
}

func (g *GithubService) CreateKey(req dto.GithubKeyCreate) (*dto.GithubKeyInfo, error) {
	key, err := githubCreateKeyRecord(strings.TrimSpace(req.Name), req.Type, nil)
	if err != nil {
		return nil, err
	}
	setting := loadGithubSetting()
	if key.Type == dto.GithubKeyTypeShared && setting.SharedKeyID == 0 {
		if err := settingRepo.UpdateOrCreate(githubSettingSharedKeyID, strconv.Itoa(int(key.ID))); err == nil {
			setting.SharedKeyID = key.ID
		}
	}
	info := githubKeyInfo(key, setting, nil)
	return &info, nil
}

func (g *GithubService) ImportKey(req dto.GithubKeyImport) (*dto.GithubKeyInfo, error) {
	key, err := githubCreateKeyRecord(strings.TrimSpace(req.Name), dto.GithubKeyTypeCustom, []byte(req.PrivateKey))
	if err != nil {
		return nil, err
	}
	info := githubKeyInfo(key, loadGithubSetting(), nil)
	return &info, nil
}

func (g *GithubService) RegenerateKey(id uint) (*dto.GithubKeyInfo, error) {
	key, err := githubRepo.GetKey(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New("ErrGithubKeyNotFound")
	}
	if key.Type == dto.GithubKeyTypeCustom {
		return nil, buserr.New("ErrGithubKeyCustomRegenerate")
	}
	pair, err := githubutil.GenerateEd25519(githubKeyComment(key.Name))
	if err != nil {
		return nil, err
	}
	if err := githubutil.WritePrivateKey(key.KeyPath, pair.PrivatePEM); err != nil {
		return nil, err
	}
	if err := githubRepo.UpdateKey(key.ID, map[string]interface{}{"public_key": pair.PublicKey, "fingerprint": pair.Fingerprint}); err != nil {
		return nil, err
	}
	key.PublicKey, key.Fingerprint = pair.PublicKey, pair.Fingerprint
	projects, _ := githubRepo.ListProjects()
	info := githubKeyInfo(key, loadGithubSetting(), projects)
	return &info, nil
}

func (g *GithubService) DeleteKey(id uint) error {
	key, err := githubRepo.GetKey(repo.WithByID(id))
	if err != nil {
		return buserr.New("ErrGithubKeyNotFound")
	}
	setting := loadGithubSetting()
	projects, _ := githubRepo.ListProjects()
	if used := githubKeyUsers(key, setting, projects); len(used) != 0 {
		return buserr.WithName("ErrGithubKeyInUse", strings.Join(used, ", "))
	}
	if setting.SharedKeyID == key.ID {
		_ = settingRepo.UpdateOrCreate(githubSettingSharedKeyID, "0")
	}
	githubDropKey(key.ID)
	return nil
}

func (g *GithubService) ListKeyOptions() ([]dto.GithubKeyOption, error) {
	keys, err := githubRepo.ListKeys(repo.WithOrderDesc("created_at"))
	if err != nil {
		return nil, err
	}
	options := make([]dto.GithubKeyOption, 0, len(keys))
	for _, key := range keys {
		options = append(options, dto.GithubKeyOption{ID: key.ID, Name: key.Name, Type: key.Type, Fingerprint: key.Fingerprint, PublicKey: key.PublicKey})
	}
	return options, nil
}

// githubCreateKeyRecord generates (privatePEM == nil) or imports a key, writes
// it to disk and stores the record.
func githubCreateKeyRecord(name, keyType string, privatePEM []byte) (model.GithubKey, error) {
	if name == "" {
		return model.GithubKey{}, buserr.New("ErrGithubInvalidName")
	}
	var (
		pair githubutil.KeyPair
		err  error
	)
	if privatePEM == nil {
		pair, err = githubutil.GenerateEd25519(githubKeyComment(name))
	} else {
		pair, err = githubutil.ParsePrivateKey(privatePEM, githubKeyComment(name))
		if errors.Is(err, githubutil.ErrKeyPassphrase) {
			return model.GithubKey{}, buserr.New("ErrGithubKeyPassphrase")
		}
		if err != nil {
			return model.GithubKey{}, buserr.New("ErrGithubInvalidKey")
		}
	}
	if err != nil {
		return model.GithubKey{}, err
	}
	key := model.GithubKey{Name: name, Type: keyType, PublicKey: pair.PublicKey, Fingerprint: pair.Fingerprint, KeyPath: "pending"}
	if err := githubRepo.CreateKey(&key); err != nil {
		return model.GithubKey{}, err
	}
	key.KeyPath = path.Join(githubKeyRoot(), strconv.Itoa(int(key.ID)), "id_ed25519")
	if keyType == dto.GithubKeyTypeCustom {
		key.KeyPath = path.Join(githubKeyRoot(), strconv.Itoa(int(key.ID)), "id_custom")
	}
	if err := githubutil.WritePrivateKey(key.KeyPath, pair.PrivatePEM); err != nil {
		_ = githubRepo.DeleteKey(repo.WithByID(key.ID))
		return model.GithubKey{}, err
	}
	if err := githubRepo.UpdateKey(key.ID, map[string]interface{}{"key_path": key.KeyPath}); err != nil {
		_ = os.RemoveAll(path.Dir(key.KeyPath))
		_ = githubRepo.DeleteKey(repo.WithByID(key.ID))
		return model.GithubKey{}, err
	}
	return key, nil
}

// githubDropKey removes a key record and its files. Missing keys are ignored.
func githubDropKey(id uint) {
	if id == 0 {
		return
	}
	key, err := githubRepo.GetKey(repo.WithByID(id))
	if err != nil {
		return
	}
	if key.KeyPath != "" && key.KeyPath != "pending" && strings.HasPrefix(key.KeyPath, githubKeyRoot()+"/") {
		_ = os.RemoveAll(path.Dir(key.KeyPath))
	}
	_ = githubRepo.DeleteKey(repo.WithByID(id))
}

// githubProjectKey returns the key a project uses, or an empty key for none.
func githubProjectKey(project model.GithubProject, setting dto.GithubSetting) (model.GithubKey, error) {
	switch project.KeyMode {
	case dto.GithubKeyModeNone:
		return model.GithubKey{}, nil
	case dto.GithubKeyModeShared:
		if setting.SharedKeyID == 0 {
			return model.GithubKey{}, buserr.New("ErrGithubSharedKeyNotSet")
		}
		return githubRepo.GetKey(repo.WithByID(setting.SharedKeyID))
	default:
		if project.KeyID == 0 {
			return model.GithubKey{}, buserr.New("ErrGithubKeyNotFound")
		}
		return githubRepo.GetKey(repo.WithByID(project.KeyID))
	}
}

func githubKeyComment(name string) string {
	clean := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return '-'
		}
		return r
	}, name)
	return "1panel-" + clean
}

func githubKeyUsers(key model.GithubKey, setting dto.GithubSetting, projects []model.GithubProject) []string {
	var used []string
	for _, project := range projects {
		switch {
		case project.KeyMode == dto.GithubKeyModeShared && setting.SharedKeyID == key.ID:
			used = append(used, project.Name)
		case (project.KeyMode == dto.GithubKeyModeProject || project.KeyMode == dto.GithubKeyModeCustom) && project.KeyID == key.ID:
			used = append(used, project.Name)
		}
	}
	return used
}

func githubKeyInfo(key model.GithubKey, setting dto.GithubSetting, projects []model.GithubProject) dto.GithubKeyInfo {
	return dto.GithubKeyInfo{
		ID:          key.ID,
		CreatedAt:   key.CreatedAt,
		Name:        key.Name,
		Type:        key.Type,
		PublicKey:   key.PublicKey,
		Fingerprint: key.Fingerprint,
		IsShared:    setting.SharedKeyID == key.ID,
		UsedBy:      githubKeyUsers(key, setting, projects),
	}
}
