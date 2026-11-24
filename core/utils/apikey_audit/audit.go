package apikey_audit

import (
	"errors"
	"fmt"
	"strings"

	"github.com/1Panel-dev/1Panel/core/app/model"
	"github.com/1Panel-dev/1Panel/core/app/repo"
	"github.com/1Panel-dev/1Panel/core/constant"
	"github.com/1Panel-dev/1Panel/core/init/session/psession"
	"github.com/1Panel-dev/1Panel/core/utils/common"
	"github.com/gin-gonic/gin"
)

type Event struct {
	KeyID     string
	KeyName   string
	Action    string
	User      string
	Status    string
	Message   string
	ClientIP  string
	OwnerType string
	OwnerID   string
}

func Record(c *gin.Context, event Event) error {
	record, err := Build(c, event)
	if err != nil {
		return err
	}
	if err := repo.NewILogRepo().CreateOperationLog(record); err != nil {
		return err
	}
	if c != nil {
		c.Set("API_KEY_AUDIT_RECORDED", true)
	}
	return nil
}

func Build(c *gin.Context, event Event) (*model.OperationLog, error) {
	labels := map[string]string{
		"create":      "Create API key",
		"update":      "Update API key",
		"enable":      "Enable API key",
		"disable":     "Disable API key",
		"revoke":      "Revoke API key",
		"delete":      "Delete API key",
		"reveal":      "Reveal API key",
		"app_binding": "Deliver API key to APP",
	}
	label, ok := labels[event.Action]
	if !ok || event.KeyID == "" || (event.Status != constant.StatusSuccess && event.Status != constant.StatusFailed) {
		return nil, errors.New("invalid API key audit event")
	}
	if event.KeyID == "legacy" {
		if event.OwnerType == "" || event.OwnerID == "" {
			return nil, errors.New("legacy audit event requires credential owner")
		}
		event.KeyID = "legacy:" + event.OwnerType + ":" + event.OwnerID
	}
	record := &model.OperationLog{
		Source: "auth", User: event.User,
		AuthMethod: "session", Status: event.Status, Message: event.Message,
		DetailZH: fmt.Sprintf("%s [%s] (%s)", label, event.KeyName, event.KeyID),
		DetailEN: fmt.Sprintf("%s [%s] (%s)", label, event.KeyName, event.KeyID),
	}
	if event.Action == "app_binding" {
		record.AuthMethod = "qr_exchange"
	}
	if c != nil {
		if record.User == "" {
			if value, exists := c.Get(psession.GinContextSessionUserKey); exists {
				if user, ok := value.(psession.SessionUser); ok {
					record.User = user.Name
				}
			}
		}
		if c.Request != nil {
			record.Path = c.Request.URL.Path
			record.Method = strings.ToLower(c.Request.Method)
			record.UserAgent = c.Request.UserAgent()
			record.IP = common.AuditClientIP(c)
		}
		if c.GetBool("API_AUTH") && event.Action != "app_binding" {
			record.AuthMethod = "api_key"
			record.APIKeyID = c.GetString("API_AUTH_KEY_ID")
			record.APIKeyName = c.GetString("API_AUTH_KEY_NAME")
			if c.GetString("API_AUTH_KEY_KIND") == "legacy" {
				record.APIKeyID = "legacy:" + c.GetString("API_AUTH_OWNER_TYPE") + ":" + c.GetString("API_AUTH_OWNER_ID")
			}
		}
	}
	if event.ClientIP != "" {
		record.IP = event.ClientIP
	}
	return record, nil
}
