// Package github holds the pure logic of the fork updater: fork version tags
// and release checksum parsing. It has no 1Panel imports and is unit tested.
package github

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Version is a fork version: upstream major.minor.patch plus the fork revision.
type Version [4]int

var (
	tagPattern  = regexp.MustCompile(`^v(\d{1,4})\.(\d{1,4})\.(\d{1,4})\.(\d{1,6})$`)
	repoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	hexPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Parse reads a strict fork tag like "v2.5.1.1". Anything else is rejected.
func Parse(tag string) (Version, bool) {
	m := tagPattern.FindStringSubmatch(strings.TrimSpace(tag))
	if m == nil {
		return Version{}, false
	}
	var v Version
	for i := 0; i < 4; i++ {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return Version{}, false
		}
		v[i] = n
	}
	return v, true
}

// Compare returns -1, 0 or 1.
func (v Version) Compare(o Version) int {
	for i := 0; i < 4; i++ {
		if v[i] < o[i] {
			return -1
		}
		if v[i] > o[i] {
			return 1
		}
	}
	return 0
}

// String formats the version as a tag.
func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d.%d", v[0], v[1], v[2], v[3])
}

// Base returns the upstream version, for example "v2.5.1" for "v2.5.1.3".
func (v Version) Base() string {
	return fmt.Sprintf("v%d.%d.%d", v[0], v[1], v[2])
}

// Newer reports whether candidate is a valid tag strictly newer than current.
// An invalid current version counts as older than any valid candidate.
func Newer(candidate, current string) bool {
	c, ok := Parse(candidate)
	if !ok {
		return false
	}
	cur, ok := Parse(current)
	if !ok {
		return true
	}
	return c.Compare(cur) > 0
}

// Base returns the upstream version of a fork tag, or "" when tag is invalid.
func Base(tag string) string {
	v, ok := Parse(tag)
	if !ok {
		return ""
	}
	return v.Base()
}

// ValidRepo reports whether repo looks like "owner/name".
func ValidRepo(repo string) bool {
	return repoPattern.MatchString(repo) && !strings.Contains(repo, "..")
}

// AssetName is the release package name for one architecture.
func AssetName(tag, arch string) string {
	return fmt.Sprintf("1panel-%s-linux-%s.tar.gz", tag, arch)
}

// ParseChecksums reads sha256sum output ("<hex>  <name>" per line).
func ParseChecksums(r io.Reader) map[string]string {
	sums := map[string]string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			continue
		}
		sum := strings.ToLower(fields[0])
		name := strings.TrimPrefix(fields[1], "*")
		if hexPattern.MatchString(sum) {
			sums[name] = sum
		}
	}
	return sums
}

// FileSHA256 returns the hex SHA-256 of a file.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
