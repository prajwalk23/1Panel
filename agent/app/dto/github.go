package dto

import "time"

// fork: request and response types of the GitHub section.

const (
	GithubKeyModeProject = "project"
	GithubKeyModeShared  = "shared"
	GithubKeyModeCustom  = "custom"
	GithubKeyModeNone    = "none"  // public repository, no credentials
	GithubKeyModeToken   = "token" // https with a username and an access token

	GithubKeyTypeProject = "project"
	GithubKeyTypeShared  = "shared"
	GithubKeyTypeCustom  = "custom"

	// Webhook mode of a project. The public URL always comes from the GitHub settings.
	GithubDomainDefault = "default" // webhook on
	GithubDomainNone    = "none"    // webhook off, deploy by hand

	GithubStrategyFFOnly = "ff-only"
	GithubStrategyReset  = "reset"

	GithubTriggerWebhook  = "webhook"
	GithubTriggerManual   = "manual"
	GithubTriggerRollback = "rollback"
	GithubTriggerCreate   = "create"
	GithubTriggerBoot     = "boot" // webhook script re-run after the server restarted

	GithubStatusWaiting  = "waiting"
	GithubStatusRunning  = "running"
	GithubStatusSuccess  = "success"
	GithubStatusFailed   = "failed"
	GithubStatusCanceled = "canceled"
	GithubStatusSkipped  = "skipped"

	// Git source of a runtime (fork). "none" means the code directory is managed
	// by hand; "git" clones and pulls from GitCloneURL.
	RuntimeGitModeNone = "none"
	RuntimeGitModeGit  = "git"
)

type GithubProjectSearch struct {
	PageInfo
	Info       string `json:"info"`
	Standalone bool   `json:"standalone"` // only projects that are not linked to a website or a runtime
}

// GithubLinkedSearch finds the project of a website or a runtime.
type GithubLinkedSearch struct {
	WebsiteID uint `json:"websiteID"`
	RuntimeID uint `json:"runtimeID"`
}

type GithubProjectOperate struct {
	ID             uint   `json:"id"`
	Name           string `json:"name" validate:"required,max=128"`
	CloneURL       string `json:"cloneURL" validate:"required,max=512"`
	Repo           string `json:"repo" validate:"max=256"`
	Branch         string `json:"branch" validate:"required,max=200"`
	ProjectPath    string `json:"projectPath" validate:"required,max=1024"`
	DeployCommand  string `json:"deployCommand" validate:"max=65535"` // legacy: scripts are now created on demand
	UpdateStrategy string `json:"updateStrategy" validate:"omitempty,oneof=ff-only reset"`
	Timeout        int    `json:"timeout" validate:"min=0,max=86400"`
	KeyMode        string `json:"keyMode" validate:"required,oneof=project shared custom none token"`
	KeyID          uint   `json:"keyID"`
	TokenUser      string `json:"tokenUser" validate:"max=128"`
	Token          string `json:"token" validate:"max=1024"` // empty on update keeps the stored token
	KeepRecords    int    `json:"keepRecords" validate:"min=0,max=100"`
	IgnorePaths    string `json:"ignorePaths" validate:"max=4096"`
	Enabled        bool   `json:"enabled"`
}

// GithubProjectDeploy runs a project now. ScriptID 0 means the script the webhook is linked to.
type GithubProjectDeploy struct {
	ID       uint   `json:"id" validate:"required"`
	ScriptID uint   `json:"scriptID"`
	Commit   string `json:"commit" validate:"max=64"` // empty: latest of the branch
	Rollback bool   `json:"rollback"`
}

type GithubCommitSearch struct {
	PageInfo
	ID     uint   `json:"id" validate:"required"`
	Branch string `json:"branch" validate:"max=200"`
}

type GithubCommitInfo struct {
	Commit  string    `json:"commit"`
	Author  string    `json:"author"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	Active  bool      `json:"active"`
}

// GithubProjectSummary feeds the Repository tab of the Git Manager.
type GithubProjectSummary struct {
	Project       GithubProjectInfo `json:"project"`
	Commit        string            `json:"commit"`
	CommitMessage string            `json:"commitMessage"`
	CommitAuthor  string            `json:"commitAuthor"`
	DeployedAt    time.Time         `json:"deployedAt"`
	Duration      int64             `json:"duration"`
}

// GithubWebhookConfig turns the webhook of a project on or off and links it to one script.
type GithubWebhookConfig struct {
	ID       uint `json:"id" validate:"required"`
	Enabled  bool `json:"enabled"`
	ScriptID uint `json:"scriptID"`
}

type GithubScriptList struct {
	ProjectID uint `json:"projectID" validate:"required"`
}

type GithubScriptOperate struct {
	ID        uint   `json:"id"`
	ProjectID uint   `json:"projectID" validate:"required"`
	Name      string `json:"name" validate:"required,max=128"`
	Content   string `json:"content" validate:"max=65535"`
	RunAsUser string `json:"runAsUser" validate:"max=64"`
}

type GithubScriptInfo struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	ProjectID uint      `json:"projectID"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	RunAsUser string    `json:"runAsUser"`
	IsDefault bool      `json:"isDefault"`
	Linked    bool      `json:"linked"`
}

type GithubProjectStatus struct {
	ID      uint `json:"id" validate:"required"`
	Enabled bool `json:"enabled"`
}

type GithubProjectInfo struct {
	ID              uint      `json:"id"`
	CreatedAt       time.Time `json:"createdAt"`
	Name            string    `json:"name"`
	Repo            string    `json:"repo"`
	CloneURL        string    `json:"cloneURL"`
	Branch          string    `json:"branch"`
	ProjectPath     string    `json:"projectPath"`
	UpdateStrategy  string    `json:"updateStrategy"`
	Timeout         int       `json:"timeout"`
	KeyMode         string    `json:"keyMode"`
	KeyID           uint      `json:"keyID"`
	KeyName         string    `json:"keyName"`
	PublicKey       string    `json:"publicKey"`
	WebhookEnabled  bool      `json:"webhookEnabled"`
	WebhookScriptID uint      `json:"webhookScriptID"`
	WebsiteID       uint      `json:"websiteID"`
	RuntimeID       uint      `json:"runtimeID"`
	TokenUser       string    `json:"tokenUser"`
	HasToken        bool      `json:"hasToken"`
	KeepRecords     int       `json:"keepRecords"`
	IgnorePaths     string    `json:"ignorePaths"`
	Enabled         bool      `json:"enabled"`
	LastStatus      string    `json:"lastStatus"`
	LastRunAt       time.Time `json:"lastRunAt"`
	LastCommit      string    `json:"lastCommit"`
	Running         bool      `json:"running"`
	WebhookURL      string    `json:"webhookURL"`
	WebhookHTTPS    bool      `json:"webhookHTTPS"`
}

// GithubProjectSecret is returned when the webhook is switched on or its token is refreshed.
type GithubProjectSecret struct {
	ID         uint   `json:"id"`
	WebhookURL string `json:"webhookURL"`
	PublicKey  string `json:"publicKey"`
}

// GithubWebhookInfo describes the webhook of a project. The URL is built on the panel address:
// the bound domain when there is one, otherwise the server IP. The browser may replace the host
// with the one the panel is opened on (Path is the part that never changes).
type GithubWebhookInfo struct {
	Enabled     bool   `json:"enabled"`
	ScriptID    uint   `json:"scriptID"`
	ScriptName  string `json:"scriptName"`
	URL         string `json:"url"`
	Path        string `json:"path"`
	BindDomain  bool   `json:"bindDomain"`
	HTTPS       bool   `json:"https"`
	ContentType string `json:"contentType"`
	Events      string `json:"events"`
}

type GithubLogSearch struct {
	PageInfo
	ProjectID uint   `json:"projectID"`
	Status    string `json:"status"`
	Trigger   string `json:"trigger"`
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

	CommitMessage string `json:"commitMessage"`
	CommitAuthor  string `json:"commitAuthor"`
	Duration      int64  `json:"duration"`
	Active        bool   `json:"active"`       // the version that is deployed now
	Rollbackable  bool   `json:"rollbackable"` // a kept version that can be rolled back to
}

type GithubLogClean struct {
	ProjectID uint   `json:"projectID"`
	Trigger   string `json:"trigger"`
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
	PublicKey   string `json:"publicKey"`
}

type GithubSetting struct {
	DefaultShell string `json:"defaultShell" validate:"required,oneof=bash sh"`
	LogRetention int    `json:"logRetention" validate:"min=1,max=10000"`
	SharedKeyID  uint   `json:"sharedKeyID"`
}

type GithubSettingInfo struct {
	GithubSetting
	WebhookBase  string `json:"webhookBase"` // panel address the webhook URLs start with
	BindDomain   bool   `json:"bindDomain"`
	GitInstalled bool   `json:"gitInstalled"`
}

// GithubRepoCheck asks whether a repository can be reached with the chosen key.
type GithubRepoCheck struct {
	CloneURL  string `json:"cloneURL" validate:"required,max=512"`
	KeyMode   string `json:"keyMode" validate:"required,oneof=project shared custom none token"`
	KeyID     uint   `json:"keyID"`
	TokenUser string `json:"tokenUser" validate:"max=128"`
	Token     string `json:"token" validate:"max=1024"`
	ProjectID uint   `json:"projectID"` // reuse the stored token of this project when Token is empty
}

type GithubRepoCheckInfo struct {
	OK            bool     `json:"ok"`
	Message       string   `json:"message"`
	DefaultBranch string   `json:"defaultBranch"`
	Branches      []string `json:"branches"`
}
