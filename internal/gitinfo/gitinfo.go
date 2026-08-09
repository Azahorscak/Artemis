// Package gitinfo collects git metadata (commit, branch, dirty flag).
package gitinfo

import (
	"os"
	"os/exec"
	"strings"
)

// Commit provenance values reported in Info.CommitSource.
const (
	// SourceUnknown means no commit could be determined at all.
	SourceUnknown = "unknown"
	// SourceGit means git itself reported the commit for the tree being built.
	SourceGit = "git"
	// SourceEnvPrefix prefixes the name of the environment variable the commit
	// was taken from, e.g. "env:GITHUB_SHA". Such a value is unverified: nothing
	// confirms it describes the tree actually being built.
	SourceEnvPrefix = "env:"
)

// commitEnvVars are consulted, in order, when git does not report a commit.
var commitEnvVars = []string{"GIT_COMMIT", "GITHUB_SHA"}

// Info holds the collected git metadata.
type Info struct {
	Commit string // full SHA-1 hash of HEAD, or empty if not available
	// CommitSource records where Commit came from, so a consumer can tell a
	// commit git vouched for from one an environment variable merely asserted.
	CommitSource string
	Branch       string // symbolic branch name (e.g. "main"); empty on detached HEAD or when git is absent
	// Dirty reports whether the working tree contains uncommitted changes, and
	// is nil when that could not be determined — git missing, dir not a
	// repository, or git erroring. Nil is deliberately distinct from a pointer
	// to false: only the latter means "inspected, and clean".
	Dirty *bool
}

// Collect gathers git metadata from the repository at dir.
// It shells out to git for commit SHA, branch name, and dirty-tree detection.
// When git is unavailable or dir is not a repository, it falls back to
// environment variables GIT_COMMIT and GITHUB_SHA for the commit (recording the
// fallback in CommitSource) and leaves Branch empty and Dirty nil.
func Collect(dir string) Info {
	info := Info{CommitSource: SourceUnknown}

	// Full commit SHA — used as a reproducibility anchor in metadata.json.
	commit, err := gitCmd(dir, "rev-parse", "HEAD")
	if err == nil {
		info.Commit = commit
		info.CommitSource = SourceGit
	}

	// Human-readable branch name; --abbrev-ref emits "HEAD" on detached HEAD state.
	branch, err := gitCmd(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		info.Branch = branch
	}

	// --porcelain produces output only when there are staged or unstaged changes.
	// Dirty is set only when the command actually succeeded: an error tells us
	// nothing about the tree, and recording "clean" there would put a claim in
	// metadata.json that was never checked.
	porcelain, err := gitCmd(dir, "status", "--porcelain")
	if err == nil {
		dirty := porcelain != ""
		info.Dirty = &dirty
	}

	// Fall back to environment variables when git didn't produce a commit. The
	// value is unverified, so record which variable supplied it.
	if info.Commit == "" {
		for _, name := range commitEnvVars {
			if v := os.Getenv(name); v != "" {
				info.Commit = v
				info.CommitSource = SourceEnvPrefix + name
				break
			}
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
