package github

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	KeyDirPerm  os.FileMode = 0o700
	KeyFilePerm os.FileMode = 0o600
)

var ErrKeyPassphrase = errors.New("passphrase-protected keys are not supported")

// KeyPair is an OpenSSH private key plus its authorized_keys public line.
type KeyPair struct {
	PrivatePEM  []byte
	PublicKey   string
	Fingerprint string
}

// GenerateEd25519 creates a new ed25519 key pair in OpenSSH format.
func GenerateEd25519(comment string) (KeyPair, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, err
	}
	block, err := ssh.MarshalPrivateKey(priv, comment)
	if err != nil {
		return KeyPair{}, err
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{
		PrivatePEM:  pem.EncodeToMemory(block),
		PublicKey:   authorizedLine(signer.PublicKey(), comment),
		Fingerprint: ssh.FingerprintSHA256(signer.PublicKey()),
	}, nil
}

// ParsePrivateKey validates a user-supplied private key and derives its public key.
func ParsePrivateKey(privatePEM []byte, comment string) (KeyPair, error) {
	data := []byte(strings.TrimSpace(string(privatePEM)) + "\n")
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		var missing *ssh.PassphraseMissingError
		if errors.As(err, &missing) {
			return KeyPair{}, ErrKeyPassphrase
		}
		return KeyPair{}, err
	}
	return KeyPair{
		PrivatePEM:  data,
		PublicKey:   authorizedLine(signer.PublicKey(), comment),
		Fingerprint: ssh.FingerprintSHA256(signer.PublicKey()),
	}, nil
}

// WritePrivateKey stores a private key with 0600 permissions in a 0700 directory.
func WritePrivateKey(path string, privatePEM []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, KeyDirPerm); err != nil {
		return err
	}
	if err := os.Chmod(dir, KeyDirPerm); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, privatePEM, KeyFilePerm); err != nil {
		return err
	}
	if err := os.Chmod(tmp, KeyFilePerm); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// EnforceKeyPerms resets permissions under root: directories 0700, files 0600.
func EnforceKeyPerms(root string) error {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	return filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Chmod(p, KeyDirPerm)
		}
		return os.Chmod(p, KeyFilePerm)
	})
}

// SSHCommand builds the GIT_SSH_COMMAND value for one key.
func SSHCommand(keyPath, knownHostsPath string) string {
	return "ssh -i " + shellQuote(keyPath) +
		" -o IdentitiesOnly=yes" +
		" -o UserKnownHostsFile=" + shellQuote(knownHostsPath) +
		" -o StrictHostKeyChecking=accept-new" +
		" -o BatchMode=yes"
}

func authorizedLine(pub ssh.PublicKey, comment string) string {
	line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
	if comment = strings.TrimSpace(comment); comment != "" {
		line += " " + comment
	}
	return line
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
