package router

// fork: fork updater routes (/api/v2/core/github/...).

import (
	v2 "github.com/1Panel-dev/1Panel/core/app/api/v2"
	"github.com/1Panel-dev/1Panel/core/middleware"
	"github.com/gin-gonic/gin"
)

type GithubRouter struct{}

func (s *GithubRouter) InitRouter(Router *gin.RouterGroup) {
	baseApi := v2.ApiGroupApp.BaseApi
	githubRouter := Router.Group("github").
		Use(middleware.SessionAuth()).
		Use(middleware.PasswordExpired())
	{
		githubRouter.GET("/fork/info", baseApi.LoadForkUpdateInfo)
		githubRouter.POST("/fork/upgrade", baseApi.ForkUpgrade)
	}
}
