package github

import (
	"strings"
	"testing"
)

const testPush = `{
  "ref": "refs/heads/main",
  "after": "0123456789abcdef0123456789abcdef01234567",
  "deleted": false,
  "repository": {"full_name": "Owner/Repo"},
  "pusher": {"name": "octocat"},
  "head_commit": {"message": "fix: thing"}
}`

func TestVerifySignature(t *testing.T) {
	body := []byte(testPush)
	secret := "s3cret"
	good := Sign(secret, body)

	if err := VerifySignature(secret, body, good); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if err := VerifySignature(secret, body, "  "+good+" "); err != nil {
		t.Fatalf("valid signature with spaces rejected: %v", err)
	}
	if err := VerifySignature("other", body, good); err != ErrSignatureInvalid {
		t.Fatalf("wrong secret: got %v", err)
	}
	if err := VerifySignature(secret, append(body, ' '), good); err != ErrSignatureInvalid {
		t.Fatalf("modified body: got %v", err)
	}
	if err := VerifySignature(secret, body, ""); err != ErrSignatureMissing {
		t.Fatalf("missing header: got %v", err)
	}
	if err := VerifySignature("", body, good); err != ErrSignatureMissing {
		t.Fatalf("empty secret: got %v", err)
	}
	if err := VerifySignature(secret, body, strings.TrimPrefix(good, "sha256=")); err != ErrSignatureInvalid {
		t.Fatalf("missing prefix: got %v", err)
	}
	if err := VerifySignature(secret, body, "sha256=zz"); err != ErrSignatureInvalid {
		t.Fatalf("bad hex: got %v", err)
	}
	if err := VerifySignature(secret, body, "sha1=abcd"); err != ErrSignatureInvalid {
		t.Fatalf("sha1 header: got %v", err)
	}
}

func TestMatch(t *testing.T) {
	push, err := ParsePush([]byte(testPush))
	if err != nil {
		t.Fatal(err)
	}
	if push.Pusher.Name != "octocat" || push.HeadCommit == nil || push.HeadCommit.Message != "fix: thing" {
		t.Fatalf("parse: %+v", push)
	}
	cases := []struct {
		name   string
		repo   string
		branch string
		event  string
		mutate func(*PushEvent)
		want   MatchResult
	}{
		{"deploy", "owner/repo", "main", EventPush, nil, MatchDeploy},
		{"ping", "owner/repo", "main", EventPing, nil, MatchPing},
		{"other event", "owner/repo", "main", "issues", nil, MatchIgnoreEvent},
		{"other branch", "owner/repo", "dev", EventPush, nil, MatchIgnoreBranch},
		{"tag push", "owner/repo", "main", EventPush, func(p *PushEvent) { p.Ref = "refs/tags/main" }, MatchIgnoreBranch},
		{"wrong repo", "owner/other", "main", EventPush, nil, MatchWrongRepo},
		{"deleted", "owner/repo", "main", EventPush, func(p *PushEvent) { p.Deleted = true }, MatchIgnoreDeleted},
		{"zero sha", "owner/repo", "main", EventPush, func(p *PushEvent) { p.After = strings.Repeat("0", 40) }, MatchIgnoreDeleted},
	}
	for _, c := range cases {
		p := push
		if c.mutate != nil {
			c.mutate(&p)
		}
		if got := Match(c.repo, c.branch, c.event, p); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestValidators(t *testing.T) {
	if !ValidSHA("0123456789abcdef0123456789abcdef01234567") {
		t.Error("sha1 rejected")
	}
	if !ValidSHA(strings.Repeat("a", 64)) {
		t.Error("sha256 rejected")
	}
	for _, bad := range []string{"", "abc", strings.Repeat("A", 40), strings.Repeat("a", 41), "$(id)" + strings.Repeat("a", 35)} {
		if ValidSHA(bad) {
			t.Errorf("sha %q accepted", bad)
		}
	}

	for _, good := range []string{"main", "release/v2", "feature-x_1.2"} {
		if !ValidBranch(good) {
			t.Errorf("branch %q rejected", good)
		}
	}
	for _, bad := range []string{"", "-x", "a..b", "a b", "a;b", "$(id)", "x.lock", "/a", "a/", "a//b"} {
		if ValidBranch(bad) {
			t.Errorf("branch %q accepted", bad)
		}
	}

	if !ValidRepo("owner/repo.js") || ValidRepo("owner") || ValidRepo("a/b/c") || ValidRepo("../x") {
		t.Error("repo validation")
	}
}

func TestRepoFromCloneURL(t *testing.T) {
	cases := map[string]string{
		"git@github.com:Owner/Repo.git":         "Owner/Repo",
		"ssh://git@github.com/owner/repo.git":   "owner/repo",
		"https://github.com/owner/repo":         "owner/repo",
		"https://github.com/owner/repo.git/":    "owner/repo",
		"https://github.com/owner":              "",
		"owner/repo":                            "",
		"https://github.com/owner/repo/tree/x":  "",
		"git@github.com:owner/repo;rm -rf.git": "",
	}
	for in, want := range cases {
		if got := RepoFromCloneURL(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
	if !IsSSHCloneURL("git@github.com:a/b.git") || IsSSHCloneURL("https://github.com/a/b") {
		t.Error("IsSSHCloneURL")
	}
}
