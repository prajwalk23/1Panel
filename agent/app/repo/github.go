package repo

import (
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/global"
	"gorm.io/gorm"
)

// fork: data access of the GitHub section.

type GithubRepo struct{}

type IGithubRepo interface {
	GetProject(opts ...DBOption) (model.GithubProject, error)
	ListProjects(opts ...DBOption) ([]model.GithubProject, error)
	PageProjects(page, size int, opts ...DBOption) (int64, []model.GithubProject, error)
	CreateProject(project *model.GithubProject) error
	SaveProject(project *model.GithubProject) error
	UpdateProject(id uint, vars map[string]interface{}) error
	DeleteProject(opts ...DBOption) error

	GetKey(opts ...DBOption) (model.GithubKey, error)
	ListKeys(opts ...DBOption) ([]model.GithubKey, error)
	PageKeys(page, size int, opts ...DBOption) (int64, []model.GithubKey, error)
	CreateKey(key *model.GithubKey) error
	UpdateKey(id uint, vars map[string]interface{}) error
	DeleteKey(opts ...DBOption) error

	GetLog(opts ...DBOption) (model.GithubDeployLog, error)
	ListLogs(opts ...DBOption) ([]model.GithubDeployLog, error)
	PageLogs(page, size int, opts ...DBOption) (int64, []model.GithubDeployLog, error)
	CreateLog(log *model.GithubDeployLog) error
	UpdateLog(id uint, vars map[string]interface{}) error
	DeleteLog(opts ...DBOption) error

	WithByProjectID(id uint) DBOption
	WithByKeyID(id uint) DBOption
	WithByWebsiteID(id uint) DBOption
	WithByLogStatus(statuses ...string) DBOption
	WithByEnabled(enabled bool) DBOption
}

func NewIGithubRepo() IGithubRepo {
	return &GithubRepo{}
}

func githubApplyOpts(db *gorm.DB, opts []DBOption) *gorm.DB {
	for _, opt := range opts {
		db = opt(db)
	}
	return db
}

func (g *GithubRepo) GetProject(opts ...DBOption) (model.GithubProject, error) {
	var item model.GithubProject
	err := githubApplyOpts(global.DB, opts).First(&item).Error
	return item, err
}

func (g *GithubRepo) ListProjects(opts ...DBOption) ([]model.GithubProject, error) {
	var items []model.GithubProject
	err := githubApplyOpts(global.DB.Model(&model.GithubProject{}), opts).Find(&items).Error
	return items, err
}

func (g *GithubRepo) PageProjects(page, size int, opts ...DBOption) (int64, []model.GithubProject, error) {
	var items []model.GithubProject
	db := githubApplyOpts(global.DB.Model(&model.GithubProject{}), opts)
	count := int64(0)
	db = db.Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Find(&items).Error
	return count, items, err
}

func (g *GithubRepo) CreateProject(project *model.GithubProject) error {
	return global.DB.Create(project).Error
}

func (g *GithubRepo) SaveProject(project *model.GithubProject) error {
	return global.DB.Save(project).Error
}

func (g *GithubRepo) UpdateProject(id uint, vars map[string]interface{}) error {
	return global.DB.Model(&model.GithubProject{}).Where("id = ?", id).Updates(vars).Error
}

func (g *GithubRepo) DeleteProject(opts ...DBOption) error {
	return githubApplyOpts(global.DB, opts).Delete(&model.GithubProject{}).Error
}

func (g *GithubRepo) GetKey(opts ...DBOption) (model.GithubKey, error) {
	var item model.GithubKey
	err := githubApplyOpts(global.DB, opts).First(&item).Error
	return item, err
}

func (g *GithubRepo) ListKeys(opts ...DBOption) ([]model.GithubKey, error) {
	var items []model.GithubKey
	err := githubApplyOpts(global.DB.Model(&model.GithubKey{}), opts).Find(&items).Error
	return items, err
}

func (g *GithubRepo) PageKeys(page, size int, opts ...DBOption) (int64, []model.GithubKey, error) {
	var items []model.GithubKey
	db := githubApplyOpts(global.DB.Model(&model.GithubKey{}), opts)
	count := int64(0)
	db = db.Count(&count)
	err := db.Order("created_at desc").Limit(size).Offset(size * (page - 1)).Find(&items).Error
	return count, items, err
}

func (g *GithubRepo) CreateKey(key *model.GithubKey) error {
	return global.DB.Create(key).Error
}

func (g *GithubRepo) UpdateKey(id uint, vars map[string]interface{}) error {
	return global.DB.Model(&model.GithubKey{}).Where("id = ?", id).Updates(vars).Error
}

func (g *GithubRepo) DeleteKey(opts ...DBOption) error {
	return githubApplyOpts(global.DB, opts).Delete(&model.GithubKey{}).Error
}

func (g *GithubRepo) GetLog(opts ...DBOption) (model.GithubDeployLog, error) {
	var item model.GithubDeployLog
	err := githubApplyOpts(global.DB, opts).First(&item).Error
	return item, err
}

func (g *GithubRepo) ListLogs(opts ...DBOption) ([]model.GithubDeployLog, error) {
	var items []model.GithubDeployLog
	err := githubApplyOpts(global.DB.Model(&model.GithubDeployLog{}), opts).Find(&items).Error
	return items, err
}

func (g *GithubRepo) PageLogs(page, size int, opts ...DBOption) (int64, []model.GithubDeployLog, error) {
	var items []model.GithubDeployLog
	db := githubApplyOpts(global.DB.Model(&model.GithubDeployLog{}), opts)
	count := int64(0)
	db = db.Count(&count)
	err := db.Order("created_at desc").Limit(size).Offset(size * (page - 1)).Find(&items).Error
	return count, items, err
}

func (g *GithubRepo) CreateLog(log *model.GithubDeployLog) error {
	return global.DB.Create(log).Error
}

func (g *GithubRepo) UpdateLog(id uint, vars map[string]interface{}) error {
	return global.DB.Model(&model.GithubDeployLog{}).Where("id = ?", id).Updates(vars).Error
}

func (g *GithubRepo) DeleteLog(opts ...DBOption) error {
	return githubApplyOpts(global.DB, opts).Delete(&model.GithubDeployLog{}).Error
}

func (g *GithubRepo) WithByProjectID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("project_id = ?", id)
	}
}

func (g *GithubRepo) WithByKeyID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("key_id = ?", id)
	}
}

func (g *GithubRepo) WithByWebsiteID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("website_id = ?", id)
	}
}

func (g *GithubRepo) WithByLogStatus(statuses ...string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("status in (?)", statuses)
	}
}

func (g *GithubRepo) WithByEnabled(enabled bool) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("enabled = ?", enabled)
	}
}
