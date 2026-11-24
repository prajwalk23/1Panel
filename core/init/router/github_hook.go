package router

// fork: public webhook endpoint of the GitHub section. A git host calls
// POST /hook?access_key=<token> on the panel address; the access key in the URL is the secret.
// The request skips login and the security entrance on purpose and is handed to the agent over
// its unix socket, which checks the key. The query string is dropped before it is forwarded so
// the key never reaches the agent's request log.

import (
	"net/http"
	"sync"
	"time"

	"github.com/1Panel-dev/1Panel/core/init/proxy"
	"github.com/1Panel-dev/1Panel/core/middleware"
	"github.com/1Panel-dev/1Panel/core/utils/common"
	"github.com/gin-gonic/gin"
)

const (
	githubHookPath       = "/hook"
	githubHookTarget     = "/internal/github/hook"
	githubHookMaxBody    = 10 << 20
	githubHookIPWindow   = time.Minute
	githubHookIPMaxCalls = 120
)

type githubHookWindow struct {
	start time.Time
	count int
}

var (
	githubHookMu   sync.Mutex
	githubHookHits = map[string]*githubHookWindow{}
)

// githubHookAllow is a small per-IP limit that keeps access key guessing slow.
func githubHookAllow(ip string) bool {
	githubHookMu.Lock()
	defer githubHookMu.Unlock()
	now := time.Now()
	if len(githubHookHits) > 4096 {
		for key, item := range githubHookHits {
			if now.Sub(item.start) > githubHookIPWindow {
				delete(githubHookHits, key)
			}
		}
	}
	item, ok := githubHookHits[ip]
	if !ok || now.Sub(item.start) > githubHookIPWindow {
		githubHookHits[ip] = &githubHookWindow{start: now, count: 1}
		return true
	}
	item.count++
	return item.count <= githubHookIPMaxCalls
}

// githubHookSourcePolicy lets the hook past the allowed IP list: the hosts that call it
// (GitHub, GitLab, ...) are not known in advance, the access key protects it.
func githubHookSourcePolicy() middleware.SourcePolicy {
	return middleware.SourcePolicy{
		Bypass: func(c *gin.Context) bool {
			return c.Request.URL.Path == githubHookPath
		},
	}
}

func registerGithubHook(r *gin.Engine) {
	r.POST(githubHookPath, proxyGithubHook)
}

func proxyGithubHook(c *gin.Context) {
	clientIP := common.GetRealClientIP(c)
	if !githubHookAllow(clientIP) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "rate limited"})
		return
	}
	key := c.Query("access_key")
	if key == "" || len(key) > 256 {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "unknown webhook"})
		return
	}
	if proxy.LocalAgentProxy == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"message": "unavailable"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, githubHookMaxBody)
	c.Request.URL.Path = githubHookTarget
	c.Request.URL.RawPath = ""
	c.Request.URL.RawQuery = ""
	c.Request.Form = nil
	c.Request.Header.Set("X-Hook-Key", key)
	c.Request.Header.Set("X-Hook-Client-IP", clientIP)
	proxy.LocalAgentProxy.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
