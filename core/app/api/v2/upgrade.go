package v2

import (
	"github.com/1Panel-dev/1Panel/core/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/gin-gonic/gin"
)

// @Tags System Setting
// @Summary Load upgrade info
// @Success 200 {object} dto.UpgradeInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/upgrade [get]
func (b *BaseApi) GetUpgradeInfo(c *gin.Context) {
	info, err := upgradeService.SearchUpgrade()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

// @Tags System Setting
// @Summary Load upgrade notes
// @Success 200 {array} dto.ReleasesNotes
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/upgrade/releases [get]
func (b *BaseApi) LoadRelease(c *gin.Context) {
	notes, err := upgradeService.LoadRelease()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, notes)
}

// @Tags System Setting
// @Summary Load release notes by version
// @Accept json
// @Param request body dto.Upgrade true "request"
// @Success 200 {string} notes
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/upgrade/notes [post]
func (b *BaseApi) GetNotesByVersion(c *gin.Context) {
	var req dto.Upgrade
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	notes, err := upgradeService.LoadNotes(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, notes)
}

// @Tags System Setting
// @Summary Upgrade
// @Accept json
// @Param request body dto.Upgrade true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/upgrade [post]
// @x-panel-log {"bodyKeys":["version"],"paramKeys":[],"BeforeFunctions":[],"formatEN":"upgrade system => [version]"}
func (b *BaseApi) Upgrade(c *gin.Context) {
	var req dto.Upgrade
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := upgradeService.Upgrade(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Load panel update plan
// @Success 200 {object} dto.UpgradePlan
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/upgrade/plan [get]
func (b *BaseApi) LoadUpgradePlan(c *gin.Context) {
	helper.SuccessWithData(c, upgradeService.LoadPlan())
}

// @Tags System Setting
// @Summary Update panel update plan
// @Accept json
// @Param request body dto.UpgradePlan true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/upgrade/plan [post]
func (b *BaseApi) UpdateUpgradePlan(c *gin.Context) {
	var req dto.UpgradePlan
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := upgradeService.UpdatePlan(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}
