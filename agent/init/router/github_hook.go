package router

// fork: internal route the panel core forwards webhook deliveries to (/hook?access_key=... on the
// panel port). It is only reachable over the agent unix socket, never from the network.

import (
	"io"
	"net"
	"net/http"

	"github.com/1Panel-dev/1Panel/agent/app/service"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/gin-gonic/gin"
)

const githubHookMaxBody = 10 << 20

func registerGithubHook(engine *gin.Engine) {
	engine.POST("/internal/github/hook", githubHookHandler)
}

func githubHookHandler(ctx *gin.Context) {
	local, ok := ctx.Request.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !global.IsMaster || !ok || local.Network() != "unix" {
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, githubHookMaxBody+1))
	if err != nil || len(body) > githubHookMaxBody {
		ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"message": "payload too large"})
		return
	}
	result := service.HandleGithubHook(service.GithubHookRequest{
		Token:    ctx.GetHeader("X-Hook-Key"),
		Event:    ctx.GetHeader("X-GitHub-Event"),
		Delivery: ctx.GetHeader("X-GitHub-Delivery"),
		ClientIP: ctx.GetHeader("X-Hook-Client-IP"),
		Body:     body,
	})
	response := gin.H{"message": result.Message}
	if result.Reason != "" {
		response["reason"] = result.Reason
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(result.Code, response)
}
