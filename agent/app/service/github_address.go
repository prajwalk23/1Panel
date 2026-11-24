package service

// fork: the webhook is served by the panel itself (/hook?access_key=...), so its URL starts
// with the panel address: the bound domain when there is one, otherwise the server IP.

import (
	"net"
	"strings"

	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/encrypt"
)

const githubHookPath = "/hook"

type githubAddress struct {
	Base       string // scheme://host[:port]
	BindDomain bool   // the host is the panel's bound domain
	HTTPS      bool
}

type githubTarget struct {
	URL        string
	Path       string
	BindDomain bool
	HTTPS      bool
}

func githubOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return ""
}

// githubPanelAddress builds the address the panel is reached on from the core settings.
func githubPanelAddress() githubAddress {
	address := githubAddress{}
	ssl, _ := settingRepo.GetCoreValueByKey("SSL")
	address.HTTPS = ssl == "Enable" || ssl == "Mux"

	host, _ := settingRepo.GetCoreValueByKey("BindDomain")
	host = strings.TrimSpace(host)
	address.BindDomain = host != ""
	if host == "" {
		host, _ = settingRepo.GetValueByKey("SystemIP")
		host = strings.TrimSpace(host)
	}
	if host == "" {
		host = githubOutboundIP()
	}
	if host == "" {
		host = "127.0.0.1"
	}
	port, _ := settingRepo.GetCoreValueByKey("ServerPort")
	port = strings.TrimSpace(port)
	if port == "" {
		port = global.CONF.Base.Port
	}

	scheme := "http"
	if address.HTTPS {
		scheme = "https"
	}
	hostPort := host
	if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		hostPort = "[" + host + "]"
	}
	if port != "" && !(scheme == "http" && port == "80") && !(scheme == "https" && port == "443") {
		hostPort += ":" + port
	}
	address.Base = scheme + "://" + hostPort
	return address
}

// githubWebhookToken returns the access key of a project, or "" when none was made.
func githubWebhookToken(project model.GithubProject) string {
	if project.WebhookToken == "" {
		return ""
	}
	token, err := encrypt.StringDecrypt(project.WebhookToken)
	if err != nil {
		return ""
	}
	return token
}

// githubWebhookTarget works out the public webhook URL of a project.
func githubWebhookTarget(project model.GithubProject) githubTarget {
	token := githubWebhookToken(project)
	if project.DomainMode == "none" || token == "" {
		return githubTarget{}
	}
	address := githubPanelAddress()
	hookPath := githubHookPath + "?access_key=" + token
	return githubTarget{URL: address.Base + hookPath, Path: hookPath, BindDomain: address.BindDomain, HTTPS: address.HTTPS}
}
