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

	WebhookSecret string `gorm:"type:text" json:"-"`

	DomainMode string `gorm:"not null;default:default" json:"domainMode"`
	WebsiteID  uint   `json:"websiteID"`
	Domain     string `json:"domain"`
	CustomURL  string `json:"customURL"`

	Enabled    bool      `json:"enabled"`
	LastStatus string    `json:"lastStatus"`
	LastRunAt  time.Time `json:"lastRunAt"`
	LastCommit string    `json:"lastCommit"`
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
}
