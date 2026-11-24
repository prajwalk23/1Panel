package router

// fork: panel memory limit routes (/api/v2/core/settings/memory-limit).

import (
	v2 "github.com/1Panel-dev/1Panel/core/app/api/v2"
	"github.com/1Panel-dev/1Panel/core/middleware"
	"github.com/gin-gonic/gin"
)

type MemoryLimitRouter struct{}

func (s *MemoryLimitRouter) InitRouter(Router *gin.RouterGroup) {
	baseApi := v2.ApiGroupApp.BaseApi
	limitRouter := Router.Group("settings").
		Use(middleware.SessionAuth()).
		Use(middleware.PasswordExpired())
	{
		limitRouter.GET("/memory-limit", baseApi.LoadPanelMemoryLimit)
		limitRouter.POST("/memory-limit", baseApi.UpdatePanelMemoryLimit)
	}
}
