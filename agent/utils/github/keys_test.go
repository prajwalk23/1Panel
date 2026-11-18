package github

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateAndParse(t *testing.T) {
	pair, err := GenerateEd25519("1panel-test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(pair.PrivatePEM), "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Fatalf("not OpenSSH format: %q", pair.PrivatePEM[:40])
	}
	if !strings.HasPrefix(pair.PublicKey, "ssh-ed25519 ") || !strings.HasSuffix(pair.PublicKey, " 1panel-test") {
		t.Fatalf("public key line: %q", pair.PublicKey)
	}
	if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pair.PublicKey)); err != nil {
		t.Fatalf("public key does not parse: %v", err)
	}
	if !strings.HasPrefix(pair.Fingerprint, "SHA256:") {
		t.Fatalf("fingerprint: %q", pair.Fingerprint)
	}

	parsed, err := ParsePrivateKey(pair.PrivatePEM, "other")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Fingerprint != pair.Fingerprint {
		t.Fatal("fingerprint changed after parse")
	}
	if _, err := ParsePrivateKey([]byte("not a key"), ""); err == nil {
		t.Fatal("garbage accepted")
	}
}

func TestWritePrivateKeyPerms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys", "7", "id_ed25519")
	pair, err := GenerateEd25519("")
	if err != nil {
		t.Fatal(err)
	}
	if err := WritePrivateKey(path, pair.PrivatePEM); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != string(pair.PrivatePEM) {
		t.Fatalf("content mismatch: %v", err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != KeyFilePerm {
		t.Fatalf("file perm %v", info.Mode().Perm())
	}
	dirInfo, _ := os.Stat(filepath.Dir(path))
	if dirInfo.Mode().Perm() != KeyDirPerm {
		t.Fatalf("dir perm %v", dirInfo.Mode().Perm())
	}

	_ = os.Chmod(path, 0o644)
	if err := EnforceKeyPerms(filepath.Join(dir, "keys")); err != nil {
		t.Fatal(err)
	}
	info, _ = os.Stat(path)
	if info.Mode().Perm() != KeyFilePerm {
		t.Fatalf("perm not enforced: %v", info.Mode().Perm())
	}
}

func TestSSHCommandQuoting(t *testing.T) {
	got := SSHCommand("/opt/1panel/github/keys/1/id_ed25519", "/opt/it's/known_hosts")
	if !strings.Contains(got, "-i '/opt/1panel/github/keys/1/id_ed25519'") {
		t.Fatalf("key not quoted: %s", got)
	}
	if !strings.Contains(got, `UserKnownHostsFile='/opt/it'\''s/known_hosts'`) {
		t.Fatalf("known hosts not quoted: %s", got)
	}
	if !strings.Contains(got, "IdentitiesOnly=yes") || !strings.Contains(got, "StrictHostKeyChecking=accept-new") {
		t.Fatalf("options missing: %s", got)
	}
}
