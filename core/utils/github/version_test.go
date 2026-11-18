package github

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAndCompare(t *testing.T) {
	good := map[string]Version{
		"v2.5.1.1":   {2, 5, 1, 1},
		"v2.10.0.12": {2, 10, 0, 12},
		" v2.5.1.2 ": {2, 5, 1, 2},
	}
	for tag, want := range good {
		got, ok := Parse(tag)
		if !ok || got != want {
			t.Errorf("Parse(%q) = %v %v", tag, got, ok)
		}
	}
	for _, bad := range []string{"", "v2.5.1", "2.5.1.1", "v2.5.1.1-beta", "v2.5.1.1.1", "v2.5.x.1", "v2.5.1.lts"} {
		if _, ok := Parse(bad); ok {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}

	ordered := []string{"v2.5.1.1", "v2.5.1.2", "v2.5.1.10", "v2.5.2.1", "v2.10.0.1", "v3.0.0.1"}
	for i := 0; i+1 < len(ordered); i++ {
		if !Newer(ordered[i+1], ordered[i]) {
			t.Errorf("%s should be newer than %s", ordered[i+1], ordered[i])
		}
		if Newer(ordered[i], ordered[i+1]) {
			t.Errorf("%s should not be newer than %s", ordered[i], ordered[i+1])
		}
	}
	if Newer("v2.5.1.1", "v2.5.1.1") {
		t.Error("equal versions reported newer")
	}
	if !Newer("v2.5.1.1", "") || Newer("garbage", "v2.5.1.1") {
		t.Error("invalid handling")
	}
}

func TestBaseAndNames(t *testing.T) {
	if Base("v2.5.1.3") != "v2.5.1" || Base("v2.5.1") != "" {
		t.Error("Base")
	}
	if AssetName("v2.5.1.1", "arm64") != "1panel-v2.5.1.1-linux-arm64.tar.gz" {
		t.Error("AssetName")
	}
	if !ValidRepo("prajwalk23/1Panel") || ValidRepo("a/b/c") || ValidRepo("../x") || ValidRepo("x") {
		t.Error("ValidRepo")
	}
}

func TestChecksums(t *testing.T) {
	sumA := strings.Repeat("a", 64)
	sumB := strings.Repeat("B", 64)
	input := sumA + "  1panel-v2.5.1.1-linux-amd64.tar.gz\n" +
		sumB + " *1panel-v2.5.1.1-linux-arm64.tar.gz\n" +
		"garbage line\n" +
		"1234  short.tar.gz\n"
	sums := ParseChecksums(strings.NewReader(input))
	if sums["1panel-v2.5.1.1-linux-amd64.tar.gz"] != sumA {
		t.Errorf("amd64 sum: %v", sums)
	}
	if sums["1panel-v2.5.1.1-linux-arm64.tar.gz"] != strings.ToLower(sumB) {
		t.Errorf("arm64 sum: %v", sums)
	}
	if _, ok := sums["short.tar.gz"]; ok || len(sums) != 2 {
		t.Errorf("unexpected entries: %v", sums)
	}

	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := FileSHA256(path)
	if err != nil || got != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Errorf("FileSHA256 = %s %v", got, err)
	}
}
