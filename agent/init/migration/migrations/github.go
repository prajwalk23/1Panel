package migrations

import (
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// fork: tables and default settings of the GitHub section. Additive only.
var AddGithubTables = &gormigrate.Migration{
	ID: "20261118-fork-add-github-tables",
	Migrate: func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.GithubProject{}, &model.GithubKey{}, &model.GithubDeployLog{}); err != nil {
			return err
		}
		defaults := []model.Setting{
			{Key: "GithubWebhookPort", Value: "19090"},
			{Key: "GithubWebhookBind", Value: "127.0.0.1"},
			{Key: "GithubDefaultShell", Value: "bash"},
			{Key: "GithubLogRetention", Value: "50"},
			{Key: "GithubDefaultWebsiteID", Value: "0"},
			{Key: "GithubDefaultDomain", Value: ""},
			{Key: "GithubCustomBaseURL", Value: ""},
			{Key: "GithubSharedKeyID", Value: "0"},
		}
		for _, item := range defaults {
			setting := item
			if err := tx.Where("key = ?", setting.Key).FirstOrCreate(&setting).Error; err != nil {
				return err
			}
		}
		return nil
	},
}
