package service

// fork: editable default pages. Next to the stock pages (404, domain404, index, php, stop)
// the 400/401/403/500/502/503/504 error pages can be edited, and every page can be reset
// to the built-in default. The error pages are served to all sites through one shared nginx
// snippet that each site's server block includes.

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	"github.com/1Panel-dev/1Panel/agent/app/dto/response"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/cmd/server/nginx_conf"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/files"
)

const (
	errorPagesDir           = "error"
	errorPagesSnippet       = "error_pages.conf"
	errorPagesContainerRoot = "/usr/share/nginx/html" // <openresty install>/root inside the container
	errorPagesURLPrefix     = "/_panel_error/"

	errorPagesDisabledSnippet = "# Custom error pages are off: nginx uses its built-in error pages.\n"
)

type errorPageDef struct {
	Code    string
	Title   string
	Message string
}

var errorPageCodes = []string{"400", "401", "403", "500", "502", "503", "504"}

var errorPageDefs = map[string]errorPageDef{
	"400": {"400", "Bad Request", "The server could not understand the request."},
	"401": {"401", "Unauthorized", "Authentication is required to view this page."},
	"403": {"403", "Forbidden", "You do not have permission to access this page."},
	"500": {"500", "Internal Server Error", "The server ran into a problem. Please try again later."},
	"502": {"502", "Bad Gateway", "The server received an invalid response from the upstream service."},
	"503": {"503", "Service Unavailable", "The service is temporarily unavailable. Please try again later."},
	"504": {"504", "Gateway Timeout", "The upstream service took too long to respond."},
}

type defaultHtmlSpec struct {
	path      string // file on the host
	content   string // built-in default
	healHan   bool   // replace the host page with the default when it contains Chinese text
}

func renderErrorPageHTML(def errorPageDef) string {
	return strings.NewReplacer(
		"{{CODE}}", def.Code,
		"{{TITLE}}", def.Title,
		"{{MESSAGE}}", def.Message,
	).Replace(string(nginx_conf.ErrorHTML))
}

func isErrorPageType(resourceType string) bool {
	_, ok := errorPageDefs[resourceType]
	return ok
}

func defaultHtmlSpecFor(rootPath, resourceType string) (defaultHtmlSpec, bool) {
	defaultPath := path.Join(rootPath, "default")
	switch resourceType {
	case "404":
		return defaultHtmlSpec{path: path.Join(defaultPath, "404.html"), content: string(nginx_conf.NotFoundHTML), healHan: true}, true
	case "php":
		return defaultHtmlSpec{path: path.Join(defaultPath, "index.php"), content: string(nginx_conf.IndexPHP), healHan: true}, true
	case "index":
		return defaultHtmlSpec{path: path.Join(defaultPath, "index.html"), content: string(nginx_conf.Index), healHan: true}, true
	case "domain404":
		return defaultHtmlSpec{path: path.Join(rootPath, "404.html"), content: string(nginx_conf.DomainNotFoundHTML), healHan: true}, true
	case "stop":
		return defaultHtmlSpec{path: path.Join(rootPath, "stop", "index.html"), content: string(nginx_conf.StopHTML), healHan: true}, true
	}
	if def, ok := errorPageDefs[resourceType]; ok {
		return defaultHtmlSpec{path: path.Join(rootPath, errorPagesDir, def.Code+".html"), content: renderErrorPageHTML(def)}, true
	}
	return defaultHtmlSpec{}, false
}

func normalizeHTML(content string) string {
	return strings.TrimSpace(strings.ReplaceAll(content, "\r\n", "\n"))
}

// hasChineseText reports whether the page contains Chinese characters. The pages the
// openresty app package ships on the host are Chinese and differ between package
// versions, so matching them by content is not reliable.
func hasChineseText(content string) bool {
	for _, r := range content {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// readDefaultHtml returns the page on the host, or the built-in default when the file is
// missing, empty or still in Chinese (which is replaced on disk).
func readDefaultHtml(fileOp files.FileOp, spec defaultHtmlSpec) *response.WebsiteHtmlRes {
	content, _ := getResourceContent(fileOp, spec.path)
	if content == "" || (spec.healHan && hasChineseText(content)) {
		if content != "" {
			_ = fileOp.SaveFile(spec.path, spec.content, constant.DirPerm)
		}
		content = spec.content
	}
	return &response.WebsiteHtmlRes{
		Content:   content,
		IsDefault: normalizeHTML(content) == normalizeHTML(spec.content),
	}
}

func getNginxRootPath() (string, error) {
	nginxInstall, err := getAppInstallByKey(constant.AppOpenresty)
	if err != nil {
		return "", err
	}
	return path.Join(nginxInstall.GetPath(), "root"), nil
}

func (w WebsiteService) ResetDefaultHtml(req request.WebsiteHtmlReset) error {
	rootPath, err := getNginxRootPath()
	if err != nil {
		return err
	}
	spec, ok := defaultHtmlSpecFor(rootPath, req.Type)
	if !ok {
		return nil
	}
	if err := files.NewFileOp().SaveFile(spec.path, spec.content, constant.DirPerm); err != nil {
		return err
	}
	if isErrorPageType(req.Type) {
		return w.EnableErrorPages()
	}
	return nil
}

// HealDefaultPages replaces default pages that are still in Chinese with the English ones.
// Files only, no nginx change; safe to run at every start.
func (w WebsiteService) HealDefaultPages() {
	rootPath, err := getNginxRootPath()
	if err != nil {
		return
	}
	fileOp := files.NewFileOp()
	for _, resourceType := range []string{"stop", "index", "php", "404", "domain404"} {
		if spec, ok := defaultHtmlSpecFor(rootPath, resourceType); ok && fileOp.Stat(spec.path) {
			readDefaultHtml(fileOp, spec)
		}
	}
}

func errorPagesInclude() string {
	return path.Join(errorPagesContainerRoot, errorPagesDir, errorPagesSnippet)
}

func errorPagesSnippetContent() string {
	var b strings.Builder
	for _, code := range errorPageCodes {
		fmt.Fprintf(&b, "error_page %s %s%s.html;\n", code, errorPagesURLPrefix, code)
	}
	fmt.Fprintf(&b, "location ^~ %s {\n    internal;\n    alias %s/%s/;\n}\n", errorPagesURLPrefix, errorPagesContainerRoot, errorPagesDir)
	return b.String()
}

// ensureErrorPageFiles writes the snippet and any missing error page. Existing pages, which
// may have been edited, and an existing snippet, which may be switched off, are left alone.
func ensureErrorPageFiles(fileOp files.FileOp, rootPath string) error {
	for _, code := range errorPageCodes {
		spec, _ := defaultHtmlSpecFor(rootPath, code)
		if !fileOp.Stat(spec.path) {
			if err := fileOp.SaveFile(spec.path, spec.content, constant.DirPerm); err != nil {
				return err
			}
		}
	}
	snippetPath := path.Join(rootPath, errorPagesDir, errorPagesSnippet)
	if current, _ := getResourceContent(fileOp, snippetPath); current == "" {
		return fileOp.SaveFile(snippetPath, errorPagesSnippetContent(), constant.DirPerm)
	}
	return nil
}

// errorPagesEnabledIn reports whether the snippet content turns the custom error pages on.
// A missing or empty snippet counts as on, because it is written with the pages when needed.
func errorPagesEnabledIn(snippet string) bool {
	return normalizeHTML(snippet) != normalizeHTML(errorPagesDisabledSnippet)
}

func errorPagesEnabled(fileOp files.FileOp, rootPath string) bool {
	snippet, _ := getResourceContent(fileOp, path.Join(rootPath, errorPagesDir, errorPagesSnippet))
	return errorPagesEnabledIn(snippet)
}

// SwitchErrorPages turns the custom error pages on or off for every site at once. Off keeps
// the shared snippet but empties it, so nginx falls back to its own built-in error pages.
func (w WebsiteService) SwitchErrorPages(req request.WebsiteErrorPagesSwitch) error {
	nginxInstall, err := getAppInstallByKey(constant.AppOpenresty)
	if err != nil {
		return err
	}
	rootPath := path.Join(nginxInstall.GetPath(), "root")
	fileOp := files.NewFileOp()
	if err := ensureErrorPageFiles(fileOp, rootPath); err != nil {
		return err
	}
	snippetPath := path.Join(rootPath, errorPagesDir, errorPagesSnippet)
	oldSnippet, _ := getResourceContent(fileOp, snippetPath)
	content := errorPagesDisabledSnippet
	if req.Enable {
		content = errorPagesSnippetContent()
	}
	if err := fileOp.SaveFile(snippetPath, content, constant.DirPerm); err != nil {
		return err
	}
	if req.Enable {
		// adds the include to sites that do not have it yet, which also reloads nginx for them
		if err := w.EnableErrorPages(); err != nil {
			_ = fileOp.SaveFile(snippetPath, oldSnippet, constant.DirPerm)
			return err
		}
	}
	return nginxCheckAndReload(oldSnippet, snippetPath, nginxInstall.ContainerName)
}

// SetSiteErrorPages makes one site follow the global custom error pages (UseGlobal) or use the
// nginx built-in ones. The site config is checked and reloaded, and restored when nginx
// rejects it, so a failed change leaves the site and its stored flag as they were.
func (w WebsiteService) SetSiteErrorPages(req request.WebsiteSiteErrorPages) error {
	website, err := websiteRepo.GetFirst(repo.WithByID(req.WebsiteID))
	if err != nil {
		return err
	}
	if website.Type == constant.Stream {
		return nil
	}
	include := errorPagesInclude()
	if req.UseGlobal {
		rootPath, err := getNginxRootPath()
		if err != nil {
			return err
		}
		if err := ensureErrorPageFiles(files.NewFileOp(), rootPath); err != nil {
			return err
		}
		content, _ := os.ReadFile(GetSitePath(website, SiteConf))
		if !strings.Contains(string(content), include) {
			params := []dto.NginxParam{{Name: "include", Params: []string{include}}}
			if err := updateNginxConfig(constant.NginxScopeServer, params, &website); err != nil {
				return err
			}
		}
	} else {
		params := []dto.NginxParam{{Name: "include", Params: []string{include}}}
		if err := deleteNginxConfig(constant.NginxScopeServer, params, &website); err != nil {
			return err
		}
	}
	website.ErrorPagesOff = !req.UseGlobal
	return websiteRepo.Save(context.Background(), &website)
}

// EnableErrorPages makes sure the pages and the shared snippet exist and that every site
// includes the snippet. A site that already includes it is not touched, so nginx is only
// reloaded for sites that need the change.
func (w WebsiteService) EnableErrorPages() error {
	rootPath, err := getNginxRootPath()
	if err != nil {
		return err
	}
	if err := ensureErrorPageFiles(files.NewFileOp(), rootPath); err != nil {
		return err
	}
	websites, err := websiteRepo.GetBy()
	if err != nil {
		return err
	}
	include := errorPagesInclude()
	for _, website := range websites {
		if website.Type == constant.Stream || website.ErrorPagesOff {
			continue
		}
		content, err := os.ReadFile(GetSitePath(website, SiteConf))
		if err != nil || strings.Contains(string(content), include) {
			continue
		}
		site := website
		params := []dto.NginxParam{{Name: "include", Params: []string{include}}}
		if err := updateNginxConfig(constant.NginxScopeServer, params, &site); err != nil {
			global.LOG.Errorf("add error pages to website %s failed, err: %v", website.Alias, err)
		}
	}
	return nil
}
