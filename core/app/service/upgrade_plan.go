package service

// Panel update plan: what to do about a new panel version.
//   none   - nothing planned, the update popup is shown
//   once   - upgrade once at a given date and time
//   daily  - upgrade to the newest version every day at a given time
//   ignore - never show the update popup until the panel is updated manually
// The plan is stored in one setting and checked by a cron job every minute, so it
// survives restarts and does not depend on an open browser tab.

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/1Panel-dev/1Panel/core/global"
	"github.com/1Panel-dev/1Panel/core/utils/common"
)

const (
	upgradePlanKey = "UpgradePlan"

	UpgradePlanNone   = "none"
	UpgradePlanOnce   = "once"
	UpgradePlanDaily  = "daily"
	UpgradePlanIgnore = "ignore"

	upgradePlanOnceLayout  = "2006-01-02 15:04"
	upgradePlanDailyLayout = "15:04"
	upgradePlanDateLayout  = "2006-01-02"
	// a planned run missed by more than this (panel was down) is skipped, not run late
	upgradePlanMaxLate = 6 * time.Hour
)

func loadUpgradePlan() dto.UpgradePlan {
	plan := dto.UpgradePlan{Mode: UpgradePlanNone}
	value, err := settingRepo.GetValueByKey(upgradePlanKey)
	if err != nil || value == "" {
		return plan
	}
	if err := json.Unmarshal([]byte(value), &plan); err != nil || plan.Mode == "" {
		return dto.UpgradePlan{Mode: UpgradePlanNone}
	}
	return plan
}

func saveUpgradePlan(plan dto.UpgradePlan) error {
	data, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	return settingRepo.UpdateOrCreate(upgradePlanKey, string(data))
}

// clearUpgradeIgnore drops the ignore flag: the panel is being updated manually.
func clearUpgradeIgnore() {
	if plan := loadUpgradePlan(); plan.Mode == UpgradePlanIgnore {
		_ = saveUpgradePlan(dto.UpgradePlan{Mode: UpgradePlanNone})
	}
}

func (u *UpgradeService) LoadPlan() dto.UpgradePlan {
	return loadUpgradePlan()
}

func (u *UpgradeService) UpdatePlan(req dto.UpgradePlan) error {
	loc := common.LoadExpiredLocation()
	plan := dto.UpgradePlan{Mode: req.Mode}
	switch req.Mode {
	case UpgradePlanNone, UpgradePlanIgnore:
	case UpgradePlanOnce:
		at, err := time.ParseInLocation(upgradePlanOnceLayout, req.Time, loc)
		if err != nil {
			return fmt.Errorf("invalid schedule time %q, expected %s", req.Time, upgradePlanOnceLayout)
		}
		if !at.After(time.Now()) {
			return fmt.Errorf("the schedule time must be in the future")
		}
		plan.Time = req.Time
	case UpgradePlanDaily:
		if _, err := time.ParseInLocation(upgradePlanDailyLayout, req.Time, loc); err != nil {
			return fmt.Errorf("invalid schedule time %q, expected %s", req.Time, upgradePlanDailyLayout)
		}
		plan.Time = req.Time
	default:
		return fmt.Errorf("unknown update plan mode %q", req.Mode)
	}
	return saveUpgradePlan(plan)
}

// RunScheduledPlan is called by the cron job every minute.
func (u *UpgradeService) RunScheduledPlan() {
	plan := loadUpgradePlan()
	if plan.Mode != UpgradePlanOnce && plan.Mode != UpgradePlanDaily {
		return
	}
	loc := common.LoadExpiredLocation()
	now := time.Now().In(loc)

	var due time.Time
	switch plan.Mode {
	case UpgradePlanOnce:
		at, err := time.ParseInLocation(upgradePlanOnceLayout, plan.Time, loc)
		if err != nil {
			_ = saveUpgradePlan(dto.UpgradePlan{Mode: UpgradePlanNone})
			return
		}
		due = at
	case UpgradePlanDaily:
		hm, err := time.ParseInLocation(upgradePlanDailyLayout, plan.Time, loc)
		if err != nil {
			_ = saveUpgradePlan(dto.UpgradePlan{Mode: UpgradePlanNone})
			return
		}
		if plan.LastRun == now.Format(upgradePlanDateLayout) {
			return
		}
		due = time.Date(now.Year(), now.Month(), now.Day(), hm.Hour(), hm.Minute(), 0, 0, loc)
	}
	if now.Before(due) {
		return
	}

	// record the run before upgrading: the upgrade restarts the panel
	if plan.Mode == UpgradePlanOnce {
		_ = saveUpgradePlan(dto.UpgradePlan{Mode: UpgradePlanNone})
	} else {
		plan.LastRun = now.Format(upgradePlanDateLayout)
		_ = saveUpgradePlan(plan)
	}
	if now.Sub(due) > upgradePlanMaxLate {
		global.LOG.Warnf("skip the scheduled panel update planned for %s, it was missed by too long", due.Format(upgradePlanOnceLayout))
		return
	}
	if status, _ := settingRepo.GetValueByKey("SystemStatus"); status == "Upgrading" {
		return
	}

	info, err := u.SearchUpgrade()
	if err != nil {
		global.LOG.Errorf("scheduled panel update: check for a new version failed: %v", err)
		return
	}
	version := info.LatestVersion
	if version == "" {
		version = info.NewVersion
	}
	if version == "" {
		global.LOG.Info("scheduled panel update: already on the newest version")
		return
	}
	global.LOG.Infof("scheduled panel update: upgrading to %s", version)
	if err := u.Upgrade(dto.Upgrade{Version: version}); err != nil {
		global.LOG.Errorf("scheduled panel update to %s failed: %v", version, err)
	}
}
