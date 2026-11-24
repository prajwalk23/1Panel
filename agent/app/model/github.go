package model

import "time"

// fork: models of the GitHub section (projects, SSH keys, deploy runs).

type GithubProject struct {
	BaseModel
	Name           string `gorm:"not null;uniqueIndex" json:"name"`
	Repo           string `gorm:"not null" json:"repo"`
	CloneURL       string `gorm:"not null" json:"cloneURL"`
	Branch         string `gorm:"not null;default:main" json:"branch"`
	ProjectPath    string `gorm:"not null" json:"projectPath"`
	DeployCommand  string `gorm:"type:text" json:"deployCommand"`
	UpdateStrategy string `gorm:"not null;default:ff-only" json:"updateStrategy"`
	RunAsUser      string `json:"runAsUser"`
	Timeout        int    `gorm:"not null;default:600" json:"timeout"`

	KeyMode string `gorm:"not null;default:project" json:"keyMode"`
	KeyID   uint   `json:"keyID"`

	WebhookSecret   string `gorm:"type:text" json:"-"`
	WebhookScriptID uint   `json:"webhookScriptID"` // the one script the webhook runs

	DomainMode string `gorm:"not null;default:default" json:"domainMode"`
	WebsiteID  uint   `json:"websiteID"`
	Domain     string `json:"domain"`
	CustomURL  string `json:"customURL"`

	Enabled    bool      `json:"enabled"`
	LastStatus string    `json:"lastStatus"`
	LastRunAt  time.Time `json:"lastRunAt"`
	LastCommit string    `json:"lastCommit"`

	// Git Manager (aaPanel style). A project is standalone, or linked to a website or a runtime.
	RuntimeID        uint   `gorm:"index" json:"runtimeID"`
	TokenUser        string `json:"tokenUser"`
	Token            string `gorm:"type:text" json:"-"`                    // https access token, encrypted
	WebhookToken     string `gorm:"type:text" json:"-"`                    // access_key of the webhook URL, encrypted
	WebhookTokenHash string `gorm:"index" json:"-"`                        // sha256 of the access_key, used for lookup
	KeepRecords      int    `gorm:"not null;default:5" json:"keepRecords"` // versions kept for rollback
	IgnorePaths      string `gorm:"type:text" json:"ignorePaths"`          // one path per line
}

// GithubScript is a named deploy script of a project. A project has several; one is the
// default and the webhook is linked to exactly one of them.
type GithubScript struct {
	BaseModel
	ProjectID uint   `gorm:"index;not null" json:"projectID"`
	Name      string `gorm:"not null" json:"name"`
	Content   string `gorm:"type:text" json:"content"`
	RunAsUser string `json:"runAsUser"`
	IsDefault bool   `json:"isDefault"`
}

type GithubKey struct {
	BaseModel
	Name        string `gorm:"not null" json:"name"`
	Type        string `gorm:"not null" json:"type"`
	PublicKey   string `gorm:"type:text;not null" json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
	KeyPath     string `gorm:"not null" json:"-"`
}

type GithubDeployLog struct {
	BaseModel
	ProjectID  uint      `gorm:"index;not null" json:"projectID"`
	Trigger    string    `gorm:"not null" json:"trigger"`
	DeliveryID string    `json:"deliveryID"`
	Ref        string    `json:"ref"`
	Commit     string    `json:"commit"`
	Pusher     string    `json:"pusher"`
	Status     string    `gorm:"not null" json:"status"`
	TaskID     string    `json:"taskID"`
	Message    string    `gorm:"type:text" json:"message"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`

	CommitMessage string `gorm:"type:text" json:"commitMessage"`
	CommitAuthor  string `json:"commitAuthor"`
	Duration      int64  `json:"duration"` // milliseconds
}
