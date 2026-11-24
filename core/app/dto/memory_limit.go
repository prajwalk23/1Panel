package dto

// fork: panel memory limit (Go soft heap limit of the core and agent processes).

type PanelProcessMemory struct {
	Running bool    `json:"running"`
	RSSMiB  float64 `json:"rssMiB"`
	HeapMiB float64 `json:"heapMiB"`
}

type PanelMemoryLimit struct {
	LimitMiB    int                `json:"limitMiB"`
	DefaultMiB  int                `json:"defaultMiB"`
	MinMiB      int                `json:"minMiB"`
	MaxMiB      int                `json:"maxMiB"`
	EnvOverride bool               `json:"envOverride"`
	Core        PanelProcessMemory `json:"core"`
	Agent       PanelProcessMemory `json:"agent"`
}

type PanelMemoryLimitUpdate struct {
	LimitMiB int `json:"limitMiB" validate:"required,min=128,max=262144"`
}
