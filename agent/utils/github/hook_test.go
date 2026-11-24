package github

import (
	"strings"
	"testing"
)

func TestTokenHelpers(t *testing.T) {
	a, err := NewToken()
	if err != nil || len(a) != 48 {
		t.Fatalf("token: %q %v", a, err)
	}
	b, _ := NewToken()
	if a == b {
		t.Fatal("tokens repeat")
	}
	if TokenHash(a) == TokenHash(b) || len(TokenHash(a)) != 64 {
		t.Fatal("hash")
	}
	if !TokenEqual(a, a) || TokenEqual(a, b) {
		t.Fatal("equal")
	}
}

func TestTokenEnv(t *testing.T) {
	if TokenEnv("u", "") != nil {
		t.Fatal("empty token must add nothing")
	}
	env := TokenEnv("octo", "ghp_x")
	joined := strings.Join(env, "\n")
	// base64("octo:ghp_x")
	if !strings.Contains(joined, "Authorization: Basic b2N0bzpnaHBfeA==") || !strings.Contains(joined, "GIT_CONFIG_COUNT=1") {
		t.Fatalf("env: %v", env)
	}
	if !strings.Contains(strings.Join(TokenEnv("", "tok"), "\n"), "x-access-token") {
		t.Fatal("default user")
	}
}

func TestMatchIgnored(t *testing.T) {
	ignore := ParseIgnorePaths("/.env\n/storage\r\n /bootstrap/cache \n\n")
	if len(ignore) != 3 || ignore[0] != ".env" || ignore[2] != "bootstrap/cache" {
		t.Fatalf("parse: %v", ignore)
	}
	cases := []struct {
		name  string
		files []string
		want  bool
	}{
		{"only ignored", []string{".env", "storage/logs/a.log", "bootstrap/cache/x.php"}, true},
		{"one real change", []string{"storage/a", "app/Models/User.php"}, false},
		{"prefix is not a match", []string{"storage-other/a"}, false},
		{"no files", nil, false},
	}
	for _, c := range cases {
		if got := MatchIgnored(ignore, c.files); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
	if MatchIgnored(nil, []string{"a"}) {
		t.Error("empty ignore list ignores nothing")
	}
	if !MatchIgnored([]string{"*.md"}, []string{"README.md"}) {
		t.Error("glob")
	}
}
