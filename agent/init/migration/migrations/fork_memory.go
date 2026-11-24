package migrations

import (
	"encoding/json"
	"errors"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// fork: GPU polling is off by default (host MonitorStatus is untouched). It can be
// switched back on in the monitor settings.
var DisableGPUMonitor = &gormigrate.Migration{
	ID: "20261124-fork-disable-gpu-monitor",
	Migrate: func(tx *gorm.DB) error {
		return tx.Model(&model.Setting{}).Where("key = ?", "GPUMonitorStatus").Update("value", constant.StatusDisable).Error
	},
}

// fork: CheckSendTimeRange refuses any alert type missing from the saved send-time-range
// config, so the new memorySustained type has to be added to it. Runs on the alert DB.
var AddMemorySustainedAlertType = &gormigrate.Migration{
	ID: "20261124-fork-add-memory-sustained-alert-type",
	Migrate: func(tx *gorm.DB) error {
		var cfgRow model.AlertConfig
		if err := tx.Where("type = ?", constant.CommonConfig).First(&cfgRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		var cfg dto.AlertCommonConfig
		if err := json.Unmarshal([]byte(cfgRow.Config), &cfg); err != nil {
			return nil
		}
		const alertType = "memorySustained"
		for _, t := range cfg.AlertSendTimeRange.NoticeAlert.Type {
			if t == alertType {
				return nil
			}
		}
		for _, t := range cfg.AlertSendTimeRange.ResourceAlert.Type {
			if t == alertType {
				return nil
			}
		}
		cfg.AlertSendTimeRange.ResourceAlert.Type = append(cfg.AlertSendTimeRange.ResourceAlert.Type, alertType)
		data, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return tx.Model(&model.AlertConfig{}).Where("id = ?", cfgRow.ID).Update("config", string(data)).Error
	},
}

// fork: the two housekeeping jobs shown in the Cron Jobs tab. Their Config marker makes
// handleSystemClean run the reduced cleanups instead of the stock full system clean.
var AddForkCleanCronjobs = &gormigrate.Migration{
	ID: "20261124-fork-add-clean-cronjobs",
	Migrate: func(tx *gorm.DB) error {
		jobs := []model.Cronjob{
			{
				Name:         "auto-clean",
				Type:         "clean",
				Spec:         "30 3 * * *",
				Status:       constant.StatusEnable,
				RetainCopies: 7,
				Timeout:      3600,
				Config:       `{"forkClean":"auto"}`,
			},
			{
				Name:         "upgrade-rollback-clean",
				Type:         "clean",
				Spec:         "40 3 * * *",
				Status:       constant.StatusEnable,
				RetainCopies: 7,
				Timeout:      3600,
				Config:       `{"forkClean":"upgradeRollback"}`,
			},
		}
		for _, item := range jobs {
			job := item
			if err := tx.Where("name = ?", job.Name).FirstOrCreate(&job).Error; err != nil {
				return err
			}
		}
		return nil
	},
}
