package v2

// fork: fork updater handlers.

import (
	"github.com/1Panel-dev/1Panel/core/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/1Panel-dev/1Panel/core/app/service"
	"github.com/gin-gonic/gin"
)

var githubUpdateService = service.NewIGithubUpdateService()

// @Tags GitHub
// @Summary Load fork update info
// @Success 200 {object} dto.ForkUpdateInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/github/fork/info [get]
func (b *BaseApi) LoadForkUpdateInfo(c *gin.Context) {
	info, err := githubUpdateService.Info()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

// @Tags GitHub
// @Summary Upgrade the panel from a fork release
// @Accept json
// @Param request body dto.ForkUpgrade true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/github/fork/upgrade [post]
func (b *BaseApi) ForkUpgrade(c *gin.Context) {
	var req dto.ForkUpgrade
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := githubUpdateService.Upgrade(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}
