package router

// fork: routes of the GitHub section (/api/v2/github/...).

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

type GithubRouter struct{}

func (s *GithubRouter) InitRouter(Router *gin.RouterGroup) {
	githubRouter := Router.Group("github")
	baseApi := v2.ApiGroupApp.BaseApi
	{
		githubRouter.POST("/projects/search", baseApi.SearchGithubProjects)
		githubRouter.POST("/projects", baseApi.CreateGithubProject)
		githubRouter.POST("/projects/update", baseApi.UpdateGithubProject)
		githubRouter.POST("/projects/del", baseApi.DeleteGithubProject)
		githubRouter.POST("/projects/status", baseApi.UpdateGithubProjectStatus)
		githubRouter.POST("/projects/deploy", baseApi.DeployGithubProject)
		githubRouter.POST("/projects/cancel", baseApi.CancelGithubDeploy)
		githubRouter.POST("/projects/linked", baseApi.LoadLinkedGithubProject)
		githubRouter.POST("/projects/summary", baseApi.LoadGithubSummary)
		githubRouter.POST("/projects/commits", baseApi.SearchGithubCommits)
		githubRouter.POST("/projects/webhook/token", baseApi.RefreshGithubWebhookToken)
		githubRouter.POST("/projects/webhook", baseApi.LoadGithubWebhook)
		githubRouter.POST("/projects/webhook/config", baseApi.ConfigureGithubWebhook)

		githubRouter.POST("/scripts/list", baseApi.ListGithubScripts)
		githubRouter.POST("/scripts/default", baseApi.LoadGithubDefaultScript)
		githubRouter.POST("/scripts", baseApi.CreateGithubScript)
		githubRouter.POST("/scripts/update", baseApi.UpdateGithubScript)
		githubRouter.POST("/scripts/del", baseApi.DeleteGithubScript)
		githubRouter.GET("/users", baseApi.ListGithubSystemUsers)

		githubRouter.POST("/logs/search", baseApi.SearchGithubLogs)
		githubRouter.POST("/logs/clean", baseApi.CleanGithubLogs)

		githubRouter.POST("/keys/search", baseApi.SearchGithubKeys)
		githubRouter.POST("/keys", baseApi.CreateGithubKey)
		githubRouter.POST("/keys/import", baseApi.ImportGithubKey)
		githubRouter.POST("/keys/regenerate", baseApi.RegenerateGithubKey)
		githubRouter.POST("/keys/del", baseApi.DeleteGithubKey)
		githubRouter.GET("/keys/options", baseApi.ListGithubKeyOptions)

		githubRouter.GET("/settings", baseApi.LoadGithubSetting)
		githubRouter.POST("/settings/update", baseApi.UpdateGithubSetting)
		githubRouter.POST("/repos/check", baseApi.CheckGithubRepo)
	}
}
