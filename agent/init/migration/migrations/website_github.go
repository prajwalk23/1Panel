package migrations

import (
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// fork: per-site opt-out of the shared custom error pages. Additive only.
var AddWebsiteErrorPagesOff = &gormigrate.Migration{
	ID: "20261125-fork-add-website-error-pages-off",
	Migrate: func(tx *gorm.DB) error {
		return tx.AutoMigrate(&model.Website{})
	},
}

// fork: GitHub source columns on websites. Additive only.
var AddWebsiteGithubFields = &gormigrate.Migration{
	ID: "20261123-fork-add-website-github-fields",
	Migrate: func(tx *gorm.DB) error {
		return tx.AutoMigrate(&model.Website{})
	},
}
