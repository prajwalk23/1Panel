package service

// fork: reduced housekeeping jobs shown in the Cron Jobs tab. Unlike the stock
// doSystemClean they never touch docker volumes, snapshot originals (1panel_original),
// the app/website/database rollback dirs or the community-restore dir.

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/task"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/i18n"
	"github.com/1Panel-dev/1Panel/agent/utils/common"
)

const (
	forkCleanAuto            = "auto"
	forkCleanUpgradeRollback = "upgradeRollback"

	forkAutoCleanLogDays = 7
)

// forkCleanMode reads the marker the seed migration stores in Cronjob.Config.
// It returns "" for user-created clean jobs, which keep the stock behaviour.
func forkCleanMode(config string) string {
	if strings.TrimSpace(config) == "" {
		return ""
	}
	var marker struct {
		ForkClean string `json:"forkClean"`
	}
	if err := json.Unmarshal([]byte(config), &marker); err != nil {
		return ""
	}
	switch marker.ForkClean {
	case forkCleanAuto, forkCleanUpgradeRollback:
		return marker.ForkClean
	}
	return ""
}

func trimVersionPrefix(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func sameVersion(a, b string) bool {
	a, b = trimVersionPrefix(a), trimVersionPrefix(b)
	return a != "" && a == b
}

func logDropSummary(taskItem *task.Task, size int64, fileCount int) {
	if fileCount != 0 {
		taskItem.Log(i18n.GetMsgWithMap("FileDropSum", map[string]interface{}{"size": common.LoadSizeUnit2F(float64(size)), "count": fileCount}))
	}
}

// doAutoClean removes temp dirs, panel logs older than 7 days and the docker build cache.
func doAutoClean(taskItem *task.Task) func(t *task.Task) error {
	return func(t *task.Task) error {
		size := int64(0)
		fileCount := 0

		dropWithTask(path.Join(global.Dir.LocalBackupDir, "tmp/system"), taskItem, &size, &fileCount)
		dropWithExclude(path.Join(global.Dir.BaseDir, uploadPath), []string{"theme", "skills-hub"}, taskItem, &size, &fileCount)
		dropWithTask(path.Join(global.Dir.BaseDir, downloadPath), taskItem, &size, &fileCount)

		cutoff := time.Now().Add(-forkAutoCleanLogDays * 24 * time.Hour)
		logFiles, _ := os.ReadDir(global.Dir.LogDir)
		for _, item := range logFiles {
			if item.IsDir() || item.Name() == "1Panel.log" || item.Name() == "1Panel-Core.log" {
				continue
			}
			info, err := item.Info()
			if err != nil || info.ModTime().After(cutoff) {
				continue
			}
			dropWithTask(path.Join(global.Dir.LogDir, item.Name()), taskItem, &size, &fileCount)
		}

		cacheCount, cacheSize := dropBuildCache()
		size += int64(cacheSize)
		fileCount += cacheCount

		logDropSummary(taskItem, size, fileCount)
		timeNow := time.Now().Format(constant.DateTimeLayout)
		_ = settingRepo.Update("LastCleanTime", timeNow)
		_ = settingRepo.Update("LastCleanSize", fmt.Sprintf("%v", size))
		_ = settingRepo.Update("LastCleanData", fmt.Sprintf("%v", fileCount))
		return nil
	}
}

type forkUpgradeLog struct {
	NewVersion string
	BackupFile string
}

// doUpgradeRollbackClean deletes the panel upgrade backups (1panel/tmp/upgrade) once the
// last recorded upgrade is the version that is actually running. A failed or rolled-back
// upgrade keeps its backups.
func doUpgradeRollbackClean(taskItem *task.Task) func(t *task.Task) error {
	return func(t *task.Task) error {
		if global.CoreDB == nil {
			taskItem.Log("core database is not available, keeping upgrade rollback files")
			return nil
		}
		var status, running model.Setting
		_ = global.CoreDB.Where("key = ?", "SystemStatus").First(&status).Error
		if status.Value == "Upgrading" {
			taskItem.Log("an upgrade is in progress, keeping upgrade rollback files")
			return nil
		}
		_ = global.CoreDB.Where("key = ?", "PanelRunningVersion").First(&running).Error
		if strings.TrimSpace(running.Value) == "" {
			taskItem.Log("running panel version is unknown yet, keeping upgrade rollback files")
			return nil
		}
		var last forkUpgradeLog
		if err := global.CoreDB.Table("upgrade_logs").
			Select("new_version, backup_file").
			Where("node_id = ?", 0).
			Order("created_at DESC").Limit(1).Scan(&last).Error; err != nil || last.NewVersion == "" {
			taskItem.Log("no panel upgrade record found, nothing to clean")
			return nil
		}
		if !sameVersion(last.NewVersion, running.Value) {
			taskItem.Log(fmt.Sprintf("last upgrade to %s is not the running version (%s), keeping upgrade rollback files", last.NewVersion, running.Value))
			return nil
		}

		size := int64(0)
		fileCount := 0
		dir := path.Join(global.Dir.BaseDir, upgradePath)
		entries, _ := os.ReadDir(dir)
		for _, entry := range entries {
			dropWithTask(path.Join(dir, entry.Name()), taskItem, &size, &fileCount)
		}
		if err := global.CoreDB.Exec("DELETE FROM upgrade_logs WHERE node_id = 0 AND backup_file LIKE ?", dir+"/%").Error; err != nil {
			taskItem.Log(fmt.Sprintf("remove upgrade records failed: %v", err))
		}
		logDropSummary(taskItem, size, fileCount)
		return nil
	}
}
