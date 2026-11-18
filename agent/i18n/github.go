package i18n

// fork: loads the GitHub section messages from lang/github/<lang>.yaml.
// The files are embedded by the existing `//go:embed lang/*` directive.

import (
	"strings"

	"github.com/1Panel-dev/1Panel/agent/global"
)

const githubLangDir = "lang/github"

func loadGithubLangFiles() {
	entries, err := fs.ReadDir(githubLangDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		file := githubLangDir + "/" + entry.Name()
		if _, err := bundle.LoadMessageFileFS(fs, file); err != nil {
			global.LOG.Errorf("[i18n] load language file %s failed: %v", file, err)
		}
	}
}
