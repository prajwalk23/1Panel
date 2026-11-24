// Package github holds the pure (DB-free) logic of the fork's GitHub section:
// webhook signature checks, push payload parsing, SSH keys, the per-project
// deploy queue and rate limiting. It is unit tested and has no 1Panel imports.
package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

const (
	EventPush = "push"
	EventPing = "ping"

	signaturePrefix = "sha256="
)

var (
	ErrSignatureMissing = errors.New("signature missing")
	ErrSignatureInvalid = errors.New("signature invalid")

	shaPattern    = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)
	repoPattern   = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	branchPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
)

// VerifySignature checks the X-Hub-Signature-256 header against an HMAC-SHA256
// of the raw body, using a constant-time compare.
func VerifySignature(secret string, body []byte, header string) error {
	header = strings.TrimSpace(header)
	if header == "" || secret == "" {
		return ErrSignatureMissing
	}
	if !strings.HasPrefix(header, signaturePrefix) {
		return ErrSignatureInvalid
	}
	got, err := hex.DecodeString(strings.TrimPrefix(header, signaturePrefix))
	if err != nil {
		return ErrSignatureInvalid
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	if !hmac.Equal(got, mac.Sum(nil)) {
		return ErrSignatureInvalid
	}
	return nil
}

// Sign returns the X-Hub-Signature-256 value for body. Used by tests and the
// panel's "send test delivery" helper.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return signaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

// PushEvent is the subset of GitHub's push payload the deploy runner needs.
type PushEvent struct {
	Ref     string `json:"ref"`
	After   string `json:"after"`
	Deleted bool   `json:"deleted"`
	Repo    struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Pusher struct {
		Name string `json:"name"`
	} `json:"pusher"`
	HeadCommit *struct {
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"head_commit"`
	Commits []struct {
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
	} `json:"commits"`
}

// ParsePush decodes a push payload.
func ParsePush(body []byte) (PushEvent, error) {
	var event PushEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return event, err
	}
	return event, nil
}

// MatchResult tells the listener what to do with a delivery.
type MatchResult int

const (
	MatchDeploy MatchResult = iota
	MatchPing
	MatchIgnoreEvent
	MatchIgnoreBranch
	MatchIgnoreDeleted
	MatchWrongRepo
)

func (m MatchResult) String() string {
	switch m {
	case MatchDeploy:
		return "deploy"
	case MatchPing:
		return "ping"
	case MatchIgnoreEvent:
		return "ignored: event"
	case MatchIgnoreBranch:
		return "ignored: branch"
	case MatchIgnoreDeleted:
		return "ignored: branch deleted"
	case MatchWrongRepo:
		return "wrong repository"
	}
	return "unknown"
}

// Match decides whether a verified delivery should start a deploy for a project
// that tracks repo (owner/name) and branch.
func Match(repo, branch, event string, push PushEvent) MatchResult {
	switch event {
	case EventPing:
		return MatchPing
	case EventPush:
	default:
		return MatchIgnoreEvent
	}
	if !strings.EqualFold(strings.TrimSpace(push.Repo.FullName), strings.TrimSpace(repo)) {
		return MatchWrongRepo
	}
	if push.Ref != "refs/heads/"+branch {
		return MatchIgnoreBranch
	}
	if push.Deleted || strings.Trim(push.After, "0") == "" {
		return MatchIgnoreDeleted
	}
	return MatchDeploy
}

// ValidSHA reports whether s is a full lowercase git object id (SHA-1 or SHA-256).
func ValidSHA(s string) bool {
	return shaPattern.MatchString(s)
}

// ValidRepo reports whether s looks like "owner/name".
func ValidRepo(s string) bool {
	return repoPattern.MatchString(s) && !strings.Contains(s, "..")
}

// ValidBranch reports whether s is a safe branch name to pass to git as an argument.
func ValidBranch(s string) bool {
	if s == "" || len(s) > 200 || strings.HasPrefix(s, "-") || strings.HasPrefix(s, "/") ||
		strings.HasSuffix(s, "/") || strings.HasSuffix(s, ".lock") || strings.Contains(s, "..") ||
		strings.Contains(s, "//") {
		return false
	}
	return branchPattern.MatchString(s)
}

// RepoFromCloneURL extracts "owner/name" from common GitHub clone URLs:
// git@github.com:owner/name.git, ssh://git@github.com/owner/name.git, https://github.com/owner/name(.git).
func RepoFromCloneURL(cloneURL string) string {
	u := strings.TrimSpace(cloneURL)
	u = strings.TrimSuffix(u, "/")
	u = strings.TrimSuffix(u, ".git")
	switch {
	case strings.HasPrefix(u, "git@"):
		if idx := strings.Index(u, ":"); idx >= 0 {
			u = u[idx+1:]
		}
	case strings.Contains(u, "://"):
		u = u[strings.Index(u, "://")+3:]
		if idx := strings.Index(u, "/"); idx >= 0 {
			u = u[idx+1:]
		} else {
			return ""
		}
	default:
		return ""
	}
	if !ValidRepo(u) {
		return ""
	}
	return u
}

// IsSSHCloneURL reports whether cloneURL needs an SSH key.
func IsSSHCloneURL(cloneURL string) bool {
	u := strings.TrimSpace(cloneURL)
	return strings.HasPrefix(u, "git@") || strings.HasPrefix(u, "ssh://")
}
