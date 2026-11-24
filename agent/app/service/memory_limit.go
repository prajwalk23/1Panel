package service

// fork: the agent follows the panel memory limit that core stores in core.db
// (setting PanelMemoryLimit, MiB). It is applied at start and re-checked every minute,
// so changing it in Settings reaches this process without a restart.

import (
	"os"
	"runtime/debug"
	"strconv"

	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/global"
)

const (
	defaultPanelMemoryLimitMiB = 512
	minPanelMemoryLimitMiB     = 128
	maxPanelMemoryLimitMiB     = 262144
)

var appliedPanelMemoryLimitMiB int

func loadPanelMemoryLimitMiB() int {
	if global.CoreDB == nil {
		return defaultPanelMemoryLimitMiB
	}
	var setting model.Setting
	if err := global.CoreDB.Where("key = ?", "PanelMemoryLimit").First(&setting).Error; err != nil {
		return defaultPanelMemoryLimitMiB
	}
	mib, err := strconv.Atoi(setting.Value)
	if err != nil || mib < minPanelMemoryLimitMiB || mib > maxPanelMemoryLimitMiB {
		return defaultPanelMemoryLimitMiB
	}
	return mib
}

// SyncPanelMemoryLimit applies the saved limit when it changed. An explicit GOMEMLIMIT
// environment variable keeps priority.
func SyncPanelMemoryLimit() {
	if os.Getenv("GOMEMLIMIT") != "" {
		return
	}
	mib := loadPanelMemoryLimitMiB()
	if mib == appliedPanelMemoryLimitMiB {
		return
	}
	debug.SetMemoryLimit(int64(mib) << 20)
	appliedPanelMemoryLimitMiB = mib
	global.LOG.Infof("panel memory limit set to %d MiB", mib)
}
