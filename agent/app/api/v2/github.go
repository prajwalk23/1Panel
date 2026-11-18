package v2

// fork: HTTP handlers of the GitHub section.

import (
	"github.com/1Panel-dev/1Panel/agent/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/service"
	"github.com/gin-gonic/gin"
)

var githubService = service.NewIGithubService()

// @Tags GitHub
// @Summary Page GitHub projects
// @Accept json
// @Param request body dto.GithubProjectSearch true "request"
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/projects/search [post]
func (b *BaseApi) SearchGithubProjects(c *gin.Context) {
	var req dto.GithubProjectSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	total, list, err := githubService.SearchProjects(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, dto.PageResult{Items: list, Total: total})
}

// @Tags GitHub
// @Summary Create GitHub project
// @Accept json
// @Param request body dto.GithubProjectOperate true "request"
// @Success 200 {object} dto.GithubProjectSecret
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/projects [post]
func (b *BaseApi) CreateGithubProject(c *gin.Context) {
	var req dto.GithubProjectOperate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	data, err := githubService.CreateProject(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags GitHub
// @Summary Update GitHub project
// @Accept json
// @Param request body dto.GithubProjectOperate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/projects/update [post]
func (b *BaseApi) UpdateGithubProject(c *gin.Context) {
	var req dto.GithubProjectOperate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if req.ID == 0 {
		helper.BadRequest(c, nil)
		return
	}
	if err := githubService.UpdateProject(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags GitHub
// @Summary Delete GitHub project
// @Accept json
// @Param request body dto.OperateByID true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/projects/del [post]
func (b *BaseApi) DeleteGithubProject(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := githubService.DeleteProject(req.ID); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags GitHub
// @Summary Enable or disable GitHub project
// @Accept json
// @Param request body dto.GithubProjectStatus true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/projects/status [post]
func (b *BaseApi) UpdateGithubProjectStatus(c *gin.Context) {
	var req dto.GithubProjectStatus
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := githubService.UpdateProjectStatus(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags GitHub
// @Summary Deploy GitHub project now
// @Accept json
// @Param request body dto.OperateByID true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/projects/deploy [post]
func (b *BaseApi) DeployGithubProject(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := githubService.Deploy(req.ID); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags GitHub
// @Summary Cancel running deploy
// @Accept json
// @Param request body dto.OperateByID true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/projects/cancel [post]
func (b *BaseApi) CancelGithubDeploy(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := githubService.Cancel(req.ID); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags GitHub
// @Summary Regenerate webhook secret
// @Accept json
// @Param request body dto.OperateByID true "request"
// @Success 200 {object} dto.GithubProjectSecret
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/projects/secret/regenerate [post]
func (b *BaseApi) RegenerateGithubSecret(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	data, err := githubService.RegenerateSecret(req.ID)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags GitHub
// @Summary Load webhook info of a project
// @Accept json
// @Param request body dto.OperateByID true "request"
// @Success 200 {object} dto.GithubWebhookInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/projects/webhook [post]
func (b *BaseApi) LoadGithubWebhook(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	data, err := githubService.LoadWebhookInfo(req.ID)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags GitHub
// @Summary Page deploy logs
// @Accept json
// @Param request body dto.GithubLogSearch true "request"
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/logs/search [post]
func (b *BaseApi) SearchGithubLogs(c *gin.Context) {
	var req dto.GithubLogSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	total, list, err := githubService.SearchLogs(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, dto.PageResult{Items: list, Total: total})
}

// @Tags GitHub
// @Summary Clean finished deploy logs
// @Accept json
// @Param request body dto.GithubLogClean true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/logs/clean [post]
func (b *BaseApi) CleanGithubLogs(c *gin.Context) {
	var req dto.GithubLogClean
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := githubService.CleanLogs(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags GitHub
// @Summary Page SSH keys
// @Accept json
// @Param request body dto.GithubKeySearch true "request"
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/keys/search [post]
func (b *BaseApi) SearchGithubKeys(c *gin.Context) {
	var req dto.GithubKeySearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	total, list, err := githubService.SearchKeys(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, dto.PageResult{Items: list, Total: total})
}

// @Tags GitHub
// @Summary Generate SSH key
// @Accept json
// @Param request body dto.GithubKeyCreate true "request"
// @Success 200 {object} dto.GithubKeyInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/keys [post]
func (b *BaseApi) CreateGithubKey(c *gin.Context) {
	var req dto.GithubKeyCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	data, err := githubService.CreateKey(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags GitHub
// @Summary Import custom SSH key
// @Accept json
// @Param request body dto.GithubKeyImport true "request"
// @Success 200 {object} dto.GithubKeyInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/keys/import [post]
func (b *BaseApi) ImportGithubKey(c *gin.Context) {
	var req dto.GithubKeyImport
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	data, err := githubService.ImportKey(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags GitHub
// @Summary Regenerate SSH key
// @Accept json
// @Param request body dto.OperateByID true "request"
// @Success 200 {object} dto.GithubKeyInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/keys/regenerate [post]
func (b *BaseApi) RegenerateGithubKey(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	data, err := githubService.RegenerateKey(req.ID)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags GitHub
// @Summary Delete SSH key
// @Accept json
// @Param request body dto.OperateByID true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/keys/del [post]
func (b *BaseApi) DeleteGithubKey(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := githubService.DeleteKey(req.ID); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags GitHub
// @Summary List selectable SSH keys
// @Success 200 {array} dto.GithubKeyOption
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/keys/options [get]
func (b *BaseApi) ListGithubKeyOptions(c *gin.Context) {
	data, err := githubService.ListKeyOptions()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags GitHub
// @Summary Load GitHub settings
// @Success 200 {object} dto.GithubSettingInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/settings [get]
func (b *BaseApi) LoadGithubSetting(c *gin.Context) {
	data, err := githubService.LoadSetting()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags GitHub
// @Summary Update GitHub settings
// @Accept json
// @Param request body dto.GithubSetting true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/settings/update [post]
func (b *BaseApi) UpdateGithubSetting(c *gin.Context) {
	var req dto.GithubSetting
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := githubService.UpdateSetting(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags GitHub
// @Summary List website domains for the webhook
// @Success 200 {array} dto.GithubDomainOption
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /github/domains [get]
func (b *BaseApi) ListGithubDomains(c *gin.Context) {
	data, err := githubService.ListDomains()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}
