package dto

import "time"

// fork: request and response types of the GitHub section.

const (
	GithubKeyModeProject = "project"
	GithubKeyModeShared  = "shared"
	GithubKeyModeCustom  = "custom"
	GithubKeyModeNone    = "none"

	GithubKeyTypeProject = "project"
	GithubKeyTypeShared  = "shared"
	GithubKeyTypeCustom  = "custom"

	GithubDomainDefault = "default"
	GithubDomainWebsite = "website"
	GithubDomainCustom  = "custom"

	GithubStrategyFFOnly = "ff-only"
	GithubStrategyReset  = "reset"

	GithubTriggerWebhook = "webhook"
	GithubTriggerManual  = "manual"

	GithubStatusWaiting  = "waiting"
	GithubStatusRunning  = "running"
	GithubStatusSuccess  = "success"
	GithubStatusFailed   = "failed"
	GithubStatusCanceled = "canceled"
	GithubStatusSkipped  = "skipped"
)

type GithubProjectSearch struct {
	PageInfo
	Info string `json:"info"`
}

type GithubProjectOperate struct {
	ID             uint   `json:"id"`
	Name           string `json:"name" validate:"required,max=128"`
	CloneURL       string `json:"cloneURL" validate:"required,max=512"`
	Repo           string `json:"repo" validate:"max=256"`
	Branch         string `json:"branch" validate:"required,max=200"`
	ProjectPath    string `json:"projectPath" validate:"required,max=1024"`
	DeployCommand  string `json:"deployCommand" validate:"max=65535"`
	UpdateStrategy string `json:"updateStrategy" validate:"required,oneof=ff-only reset"`
	RunAsUser      string `json:"runAsUser" validate:"max=64"`
	Timeout        int    `json:"timeout" validate:"min=0,max=86400"`
	KeyMode        string `json:"keyMode" validate:"required,oneof=project shared custom none"`
	KeyID          uint   `json:"keyID"`
	DomainMode     string `json:"domainMode" validate:"required,oneof=default website custom"`
	WebsiteID      uint   `json:"websiteID"`
	Domain         string `json:"domain" validate:"max=256"`
	CustomURL      string `json:"customURL" validate:"max=1024"`
	Enabled        bool   `json:"enabled"`
}

type GithubProjectStatus struct {
	ID      uint `json:"id" validate:"required"`
	Enabled bool `json:"enabled"`
}

type GithubProjectInfo struct {
	ID             uint      `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	Name           string    `json:"name"`
	Repo           string    `json:"repo"`
	CloneURL       string    `json:"cloneURL"`
	Branch         string    `json:"branch"`
	ProjectPath    string    `json:"projectPath"`
	DeployCommand  string    `json:"deployCommand"`
	UpdateStrategy string    `json:"updateStrategy"`
	RunAsUser      string    `json:"runAsUser"`
	Timeout        int       `json:"timeout"`
	KeyMode        string    `json:"keyMode"`
	KeyID          uint      `json:"keyID"`
	KeyName        string    `json:"keyName"`
	PublicKey      string    `json:"publicKey"`
	DomainMode     string    `json:"domainMode"`
	WebsiteID      uint      `json:"websiteID"`
	Domain         string    `json:"domain"`
	CustomURL      string    `json:"customURL"`
	Enabled        bool      `json:"enabled"`
	LastStatus     string    `json:"lastStatus"`
	LastRunAt      time.Time `json:"lastRunAt"`
	LastCommit     string    `json:"lastCommit"`
	Running        bool      `json:"running"`
	WebhookURL     string    `json:"webhookURL"`
	WebhookHTTPS   bool      `json:"webhookHTTPS"`
}

// GithubProjectSecret is the only response that contains a webhook secret.
type GithubProjectSecret struct {
	ID         uint   `json:"id"`
	Secret     string `json:"secret"`
	WebhookURL string `json:"webhookURL"`
	PublicKey  string `json:"publicKey"`
}

type GithubWebhookInfo struct {
	URL         string `json:"url"`
	HTTPS       bool   `json:"https"`
	ContentType string `json:"contentType"`
	Events      string `json:"events"`
	ProxyReady  bool   `json:"proxyReady"`
	Listener    string `json:"listener"`
	Message     string `json:"message"`
}

type GithubLogSearch struct {
	PageInfo
	ProjectID uint   `json:"projectID"`
	Status    string `json:"status"`
}

type GithubLogInfo struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	ProjectID   uint      `json:"projectID"`
	ProjectName string    `json:"projectName"`
	Trigger     string    `json:"trigger"`
	DeliveryID  string    `json:"deliveryID"`
	Ref         string    `json:"ref"`
	Commit      string    `json:"commit"`
	Pusher      string    `json:"pusher"`
	Status      string    `json:"status"`
	TaskID      string    `json:"taskID"`
	Message     string    `json:"message"`
	StartedAt   time.Time `json:"startedAt"`
	FinishedAt  time.Time `json:"finishedAt"`
}

type GithubLogClean struct {
	ProjectID uint `json:"projectID"`
}

type GithubKeySearch struct {
	PageInfo
	Info string `json:"info"`
}

type GithubKeyCreate struct {
	Name string `json:"name" validate:"required,max=128"`
	Type string `json:"type" validate:"required,oneof=project shared"`
}

type GithubKeyImport struct {
	Name       string `json:"name" validate:"required,max=128"`
	PrivateKey string `json:"privateKey" validate:"required,max=16384"`
}

type GithubKeyInfo struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	PublicKey   string    `json:"publicKey"`
	Fingerprint string    `json:"fingerprint"`
	IsShared    bool      `json:"isShared"`
	UsedBy      []string  `json:"usedBy"`
}

type GithubKeyOption struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
}

type GithubSetting struct {
	WebhookPort      int    `json:"webhookPort" validate:"required,min=1,max=65535"`
	WebhookBind      string `json:"webhookBind" validate:"required,max=64"`
	DefaultShell     string `json:"defaultShell" validate:"required,oneof=bash sh"`
	LogRetention     int    `json:"logRetention" validate:"min=1,max=10000"`
	DefaultWebsiteID uint   `json:"defaultWebsiteID"`
	DefaultDomain    string `json:"defaultDomain" validate:"max=256"`
	CustomBaseURL    string `json:"customBaseURL" validate:"max=1024"`
	SharedKeyID      uint   `json:"sharedKeyID"`
}

type GithubSettingInfo struct {
	GithubSetting
	ListenerRunning bool   `json:"listenerRunning"`
	ListenerAddr    string `json:"listenerAddr"`
	ListenerError   string `json:"listenerError"`
	GitInstalled    bool   `json:"gitInstalled"`
}

type GithubDomainOption struct {
	WebsiteID     uint   `json:"websiteID"`
	Alias         string `json:"alias"`
	PrimaryDomain string `json:"primaryDomain"`
	Domain        string `json:"domain"`
	Port          int    `json:"port"`
	HTTPS         bool   `json:"https"`
	HookProxy     bool   `json:"hookProxy"`
}
