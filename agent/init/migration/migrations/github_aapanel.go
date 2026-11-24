package migrations

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/utils/encrypt"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// fork: Git Manager columns (token auth, webhook access key, rollback records, ignored paths,
// runtime link). Projects whose webhook is on get an access key so the new URL works at once.
var AddGithubManager = &gormigrate.Migration{
	ID: "20261124-fork-add-github-manager",
	Migrate: func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.GithubProject{}, &model.GithubDeployLog{}); err != nil {
			return err
		}
		setting := model.Setting{Key: "GithubLastBootTime", Value: "0"}
		if err := tx.Where("key = ?", setting.Key).FirstOrCreate(&setting).Error; err != nil {
			return err
		}
		var projects []model.GithubProject
		if err := tx.Where("domain_mode <> ? AND (webhook_token_hash IS NULL OR webhook_token_hash = '')", "none").Find(&projects).Error; err != nil {
			return err
		}
		for _, project := range projects {
			buf := make([]byte, 24)
			if _, err := rand.Read(buf); err != nil {
				return err
			}
			token := hex.EncodeToString(buf)
			encrypted, err := encrypt.StringEncrypt(token)
			if err != nil {
				return err
			}
			sum := sha256.Sum256([]byte(token))
			if err := tx.Model(&model.GithubProject{}).Where("id = ?", project.ID).Updates(map[string]interface{}{
				"webhook_token":      encrypted,
				"webhook_token_hash": hex.EncodeToString(sum[:]),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	},
}
