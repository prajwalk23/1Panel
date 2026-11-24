package model

import "time"

type Website struct {
	BaseModel
	Protocol      string    `gorm:"not null" json:"protocol"`
	PrimaryDomain string    `gorm:"not null" json:"primaryDomain"`
	Type          string    `gorm:"not null" json:"type"`
	Alias         string    `gorm:"not null" json:"alias"`
	Remark        string    `json:"remark"`
	Status        string    `gorm:"not null" json:"status"`
	HttpConfig    string    `gorm:"not null" json:"httpConfig"`
	ExpireDate    time.Time `json:"expireDate"`

	Proxy         string `json:"proxy"`
	ProxyType     string `json:"proxyType"`
	SiteDir       string `json:"siteDir"`
	ErrorLog      bool   `json:"errorLog"`
	AccessLog     bool   `json:"accessLog"`
	DefaultServer bool   `json:"defaultServer"`
	IPV6          bool   `json:"IPV6"`
	Rewrite       string `json:"rewrite"`

	WebsiteGroupID  uint `json:"webSiteGroupId"`
	WebsiteSSLID    uint `json:"webSiteSSLId"`
	RuntimeID       uint `json:"runtimeID"`
	AppInstallID    uint `json:"appInstallId"`
	FtpID           uint `json:"ftpId"`
	ParentWebsiteID uint `json:"parentWebsiteID"`

	User  string `json:"user"`
	Group string `json:"group"`

	DbType string `json:"dbType"`
	DbID   uint   `json:"dbID"`

	Favorite bool `json:"favorite"`

	StreamPorts string `json:"streamPorts"`

	// fork: optional GitHub source of a PHP website (cloned into the site index dir).
	GitMode           string `gorm:"default:none" json:"gitMode"`
	GitCloneURL       string `json:"gitCloneURL"`
	GitBranch         string `gorm:"default:main" json:"gitBranch"`
	GitKeyID          uint   `json:"gitKeyID"`
	GitUpdateStrategy string `gorm:"default:ff-only" json:"gitUpdateStrategy"`
	GitAuthMode       string `gorm:"-" json:"-"` // create request only, never stored on the site
	GitTokenUser      string `gorm:"-" json:"-"`
	GitToken          string `gorm:"-" json:"-"`

	// fork: the site opts out of the shared custom error pages and uses the nginx built-in ones.
	// false (default) means the site follows the global setting.
	ErrorPagesOff bool `gorm:"default:false" json:"errorPagesOff"`

	Domains    []WebsiteDomain `json:"domains" gorm:"-:migration"`
	WebsiteSSL WebsiteSSL      `json:"webSiteSSL" gorm:"-:migration"`
}

func (w Website) TableName() string {
	return "websites"
}
