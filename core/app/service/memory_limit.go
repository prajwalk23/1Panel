package service

// fork: editable panel memory limit. The value is the Go soft heap limit (GOMEMLIMIT
// semantics) of the core process and, through the core.db setting, of the agent process.

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"

	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/shirou/gopsutil/v4/process"
)

const (
	PanelMemoryLimitKey        = "PanelMemoryLimit"
	DefaultPanelMemoryLimitMiB = 512
	MinPanelMemoryLimitMiB     = 128
	MaxPanelMemoryLimitMiB     = 262144

	agentProcessName = "1panel-agent"
	bytesPerMiB      = 1 << 20
)

// memoryLimitEnvSet reports whether a GOMEMLIMIT environment variable overrides the setting.
func memoryLimitEnvSet() bool {
	return os.Getenv("GOMEMLIMIT") != ""
}

// LoadPanelMemoryLimitMiB returns the saved limit, or the default when unset or invalid.
func LoadPanelMemoryLimitMiB() int {
	value, err := settingRepo.GetValueByKey(PanelMemoryLimitKey)
	if err != nil {
		return DefaultPanelMemoryLimitMiB
	}
	mib, err := strconv.Atoi(value)
	if err != nil || mib < MinPanelMemoryLimitMiB || mib > MaxPanelMemoryLimitMiB {
		return DefaultPanelMemoryLimitMiB
	}
	return mib
}

// ApplyPanelMemoryLimit sets the core process limit from the setting. An explicit
// GOMEMLIMIT environment variable keeps priority.
func ApplyPanelMemoryLimit() {
	if memoryLimitEnvSet() {
		return
	}
	debug.SetMemoryLimit(int64(LoadPanelMemoryLimitMiB()) * bytesPerMiB)
}

func UpdatePanelMemoryLimit(req dto.PanelMemoryLimitUpdate) error {
	if req.LimitMiB < MinPanelMemoryLimitMiB || req.LimitMiB > MaxPanelMemoryLimitMiB {
		return fmt.Errorf("memory limit must be between %d and %d MiB", MinPanelMemoryLimitMiB, MaxPanelMemoryLimitMiB)
	}
	if err := settingRepo.UpdateOrCreate(PanelMemoryLimitKey, strconv.Itoa(req.LimitMiB)); err != nil {
		return err
	}
	ApplyPanelMemoryLimit() // the agent picks the new value up within a minute
	return nil
}

func LoadPanelMemoryLimit() dto.PanelMemoryLimit {
	return dto.PanelMemoryLimit{
		LimitMiB:    LoadPanelMemoryLimitMiB(),
		DefaultMiB:  DefaultPanelMemoryLimitMiB,
		MinMiB:      MinPanelMemoryLimitMiB,
		MaxMiB:      MaxPanelMemoryLimitMiB,
		EnvOverride: memoryLimitEnvSet(),
		Core:        loadCoreMemory(),
		Agent:       loadAgentMemory(),
	}
}

func loadCoreMemory() dto.PanelProcessMemory {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	item := dto.PanelProcessMemory{Running: true, HeapMiB: float64(stats.HeapAlloc) / bytesPerMiB}
	if proc, err := process.NewProcess(int32(os.Getpid())); err == nil {
		if info, err := proc.MemoryInfo(); err == nil {
			item.RSSMiB = float64(info.RSS) / bytesPerMiB
		}
	}
	return item
}

func loadAgentMemory() dto.PanelProcessMemory {
	procs, err := process.Processes()
	if err != nil {
		return dto.PanelProcessMemory{}
	}
	for _, proc := range procs {
		name, err := proc.Name()
		if err != nil || name != agentProcessName {
			continue
		}
		info, err := proc.MemoryInfo()
		if err != nil {
			continue
		}
		return dto.PanelProcessMemory{Running: true, RSSMiB: float64(info.RSS) / bytesPerMiB}
	}
	return dto.PanelProcessMemory{}
}
