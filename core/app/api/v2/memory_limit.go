package v2

// fork: panel memory limit handlers.

import (
	"github.com/1Panel-dev/1Panel/core/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/1Panel-dev/1Panel/core/app/service"
	"github.com/gin-gonic/gin"
)

// @Tags System Setting
// @Summary Load panel memory limit and current usage
// @Success 200 {object} dto.PanelMemoryLimit
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/memory-limit [get]
func (b *BaseApi) LoadPanelMemoryLimit(c *gin.Context) {
	helper.SuccessWithData(c, service.LoadPanelMemoryLimit())
}

// @Tags System Setting
// @Summary Update panel memory limit
// @Accept json
// @Param request body dto.PanelMemoryLimitUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/memory-limit [post]
// @x-panel-log {"bodyKeys":["limitMiB"],"paramKeys":[],"BeforeFunctions":[],"formatEN":"update panel memory limit [limitMiB] MiB"}
func (b *BaseApi) UpdatePanelMemoryLimit(c *gin.Context) {
	var req dto.PanelMemoryLimitUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := service.UpdatePanelMemoryLimit(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}
