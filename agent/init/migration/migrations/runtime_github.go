package migrations

import (
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// fork: GitHub source columns on runtimes. Additive only.
var AddRuntimeGithubFields = &gormigrate.Migration{
	ID: "20261122-fork-add-runtime-github-fields",
	Migrate: func(tx *gorm.DB) error {
		return tx.AutoMigrate(&model.Runtime{})
	},
}
