// Package gitinfo collects git metadata (commit, branch, dirty flag).
package gitinfo

import (
	"os"
	"os/exec"
	"strings"
)

// Info holds the collected git metadata.
type Info struct {
	Commit string
	Branch string
	Dirty  bool
}

// Collect gathers git metadata from the repository at dir.
// It shells out to git for commit SHA, branch name, and dirty-tree detection.
// When git is unavailable or dir is not a repository, it falls back to
// environment variables GIT_COMMIT and GITHUB_SHA for the commit, and leaves
// Branch empty / Dirty false.
func Collect(dir string) Info {
	info := Info{}

	commit, err := gitCmd(dir, "rev-parse", "HEAD")
	if err == nil {
		info.Commit = commit
	}

	branch, err := gitCmd(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		info.Branch = branch
	}

	porcelain, err := gitCmd(dir, "status", "--porcelain")
	if err == nil && porcelain != "" {
		info.Dirty = true
	}

	// Fall back to environment variables when git didn't produce a commit.
	if info.Commit == "" {
		if v := os.Getenv("GIT_COMMIT"); v != "" {
			info.Commit = v
		} else if v := os.Getenv("GITHUB_SHA"); v != "" {
			info.Commit = v
		}
	}

	return info
}

// gitCmd runs a git command in dir and returns its trimmed stdout.
func gitCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
