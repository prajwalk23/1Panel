package github

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"path"
	"strings"
)

// NewToken returns a random access key for a webhook URL.
func NewToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// TokenHash is what the database keeps to find a project by its access key.
func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// TokenEqual compares two access keys in constant time.
func TokenEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// TokenEnv returns git environment variables that send a username and an access token
// to an https remote as a basic auth header. The token never reaches argv or .git/config.
func TokenEnv(user, token string) []string {
	if token == "" {
		return nil
	}
	if user == "" {
		user = "x-access-token"
	}
	credential := base64.StdEncoding.EncodeToString([]byte(user + ":" + token))
	return []string{
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.extraHeader",
		"GIT_CONFIG_VALUE_0=Authorization: Basic " + credential,
	}
}

// IsHTTPSCloneURL reports whether cloneURL is an https remote.
func IsHTTPSCloneURL(cloneURL string) bool {
	return strings.HasPrefix(strings.TrimSpace(cloneURL), "https://")
}

// ParseIgnorePaths splits the ignored path setting (one path per line) into clean entries.
func ParseIgnorePaths(raw string) []string {
	var list []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "/")
		if line != "" {
			list = append(list, line)
		}
	}
	return list
}

// ChangedFiles returns every file a push touched.
func (p PushEvent) ChangedFiles() []string {
	var files []string
	for _, commit := range p.Commits {
		files = append(files, commit.Added...)
		files = append(files, commit.Modified...)
		files = append(files, commit.Removed...)
	}
	return files
}

// MatchIgnored reports whether every changed file lies under an ignored path, so the
// push does not need a deploy. No changed files (a force push, a payload without commit
// details) is never ignored. An entry matches a file or a folder, and may use * globs.
func MatchIgnored(ignore []string, files []string) bool {
	if len(ignore) == 0 || len(files) == 0 {
		return false
	}
	for _, file := range files {
		file = strings.Trim(file, "/")
		covered := false
		for _, entry := range ignore {
			if file == entry || strings.HasPrefix(file, entry+"/") {
				covered = true
				break
			}
			if ok, err := path.Match(entry, file); err == nil && ok {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}
