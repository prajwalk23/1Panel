package service

// fork: "memorySustained" alert. Fires once memory usage has stayed at or above the
// threshold (alert.Count, %) continuously for alert.Cycle minutes. The stock "memory"
// alert averages a window of samples instead, so a short spike can trip it.

import (
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/shirou/gopsutil/v4/mem"
)

// A sample gap longer than this (cron skipped, outside the send time range, ...)
// breaks the "continuous" requirement and re-arms the timer.
const memSustainedMaxGap = 2 * time.Minute

type memSustainedState struct {
	since time.Time
	last  time.Time
}

var memSustained memSustainedState

// memSustainedStep feeds one sample into the tracker and reports whether the usage has
// been at or above threshold for at least cycleMin minutes without interruption.
func memSustainedStep(now time.Time, percent, threshold float64, cycleMin int, st *memSustainedState) bool {
	if percent < threshold {
		*st = memSustainedState{}
		return false
	}
	if st.since.IsZero() || now.Sub(st.last) > memSustainedMaxGap {
		st.since = now
	}
	st.last = now
	return now.Sub(st.since) >= time.Duration(cycleMin)*time.Minute
}

func resetMemSustainedState() {
	memSustained = memSustainedState{}
}

func loadMemSustainedUsage(alert dto.AlertDTO) {
	memStat, err := mem.VirtualMemory()
	if err != nil {
		global.LOG.Errorf("load memory usage for sustained alert failed, err: %v", err)
		return
	}
	cycle := int(alert.Cycle)
	if cycle < 1 {
		return
	}
	if !memSustainedStep(time.Now(), memStat.UsedPercent, float64(alert.Count), cycle, &memSustained) {
		return
	}
	newDate, err := alertRepo.GetTaskLog(alert.Type, alert.ID)
	if err != nil {
		global.LOG.Errorf("record not found, err: %v", err)
	}
	if isAlertDue(newDate) {
		sendResourceAlert(alert, memStat.UsedPercent)
	}
}
