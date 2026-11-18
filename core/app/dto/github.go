package dto

// fork: fork updater types.

type ForkUpdateInfo struct {
	IsForkBuild   bool   `json:"isForkBuild"`
	PanelVersion  string `json:"panelVersion"`
	ForkVersion   string `json:"forkVersion"`
	ForkRepo      string `json:"forkRepo"`
	LatestVersion string `json:"latestVersion"`
	LatestBase    string `json:"latestBase"`
	HasUpdate     bool   `json:"hasUpdate"`
	BaseChange    bool   `json:"baseChange"`
	ReleaseName   string `json:"releaseName"`
	ReleaseNote   string `json:"releaseNote"`
	ReleaseURL    string `json:"releaseURL"`
	PublishedAt   string `json:"publishedAt"`
	Arch          string `json:"arch"`
	AssetReady    bool   `json:"assetReady"`
	SystemStatus  string `json:"systemStatus"`
}

type ForkUpgrade struct {
	Version string `json:"version" validate:"required"`
}
