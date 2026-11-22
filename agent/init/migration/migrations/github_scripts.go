package migrations

import (
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// fork: deploy scripts and the webhook-to-script link. Additive only (new table, new column).
var AddGithubScripts = &gormigrate.Migration{
	ID: "20261122-fork-add-github-scripts",
	Migrate: func(tx *gorm.DB) error {
		return tx.AutoMigrate(&model.GithubScript{}, &model.GithubProject{})
	},
}
