package gitinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initRepo creates a new git repo in a temp directory, configures a dummy
// user, and returns the repo path. It does NOT create an initial commit.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	run(t, dir, "git", "config", "commit.gpgsign", "false")

	return dir
}

// commitFile writes a file and commits it, returning the full commit SHA.
func commitFile(t *testing.T, dir, name, content, msg string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", name)
	run(t, dir, "git", "commit", "-m", msg)

	out := run(t, dir, "git", "rev-parse", "HEAD")
	return out
}

// run executes a command in dir and returns trimmed stdout. It fails the test
// on error.
func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
	return string(out[:len(out)-func() int {
		if len(out) > 0 && out[len(out)-1] == '\n' {
			return 1
		}
		return 0
	}()])
}

// ---------- git-based tests ----------

func TestCollect_CleanRepo(t *testing.T) {
	dir := initRepo(t)
	sha := commitFile(t, dir, "hello.txt", "hello", "initial commit")

	info := Collect(dir)

	if info.Commit != sha {
		t.Errorf("Commit = %q, want %q", info.Commit, sha)
	}
	if info.Dirty {
		t.Error("expected clean tree, got Dirty=true")
	}
}

func TestCollect_DirtyTree(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "hello.txt", "hello", "initial commit")

	// Create an untracked modification.
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	info := Collect(dir)

	if !info.Dirty {
		t.Error("expected Dirty=true after modifying tracked file")
	}
}

func TestCollect_UntrackedFile(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "hello.txt", "hello", "initial commit")

	// Add an untracked file — porcelain should report it.
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	info := Collect(dir)

	if !info.Dirty {
		t.Error("expected Dirty=true with untracked file")
	}
}

func TestCollect_Branch(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "hello.txt", "hello", "initial commit")

	// Default branch after init+commit should be detected.
	info := Collect(dir)

	if info.Branch == "" {
		t.Error("expected non-empty Branch")
	}
}

func TestCollect_CustomBranch(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "hello.txt", "hello", "initial commit")

	run(t, dir, "git", "checkout", "-b", "feature/test-branch")

	info := Collect(dir)

	if info.Branch != "feature/test-branch" {
		t.Errorf("Branch = %q, want %q", info.Branch, "feature/test-branch")
	}
}

// ---------- env-var fallback tests ----------

func TestCollect_FallbackGIT_COMMIT(t *testing.T) {
	dir := t.TempDir() // not a git repo

	t.Setenv("GIT_COMMIT", "abc123")
	t.Setenv("GITHUB_SHA", "")

	info := Collect(dir)

	if info.Commit != "abc123" {
		t.Errorf("Commit = %q, want %q", info.Commit, "abc123")
	}
}

func TestCollect_FallbackGITHUB_SHA(t *testing.T) {
	dir := t.TempDir() // not a git repo

	t.Setenv("GIT_COMMIT", "")
	t.Setenv("GITHUB_SHA", "def456")

	info := Collect(dir)

	if info.Commit != "def456" {
		t.Errorf("Commit = %q, want %q", info.Commit, "def456")
	}
}

func TestCollect_GIT_COMMIT_TakesPrecedenceOverGITHUB_SHA(t *testing.T) {
	dir := t.TempDir() // not a git repo

	t.Setenv("GIT_COMMIT", "from-git-commit")
	t.Setenv("GITHUB_SHA", "from-github-sha")

	info := Collect(dir)

	if info.Commit != "from-git-commit" {
		t.Errorf("Commit = %q, want %q (GIT_COMMIT should take precedence)", info.Commit, "from-git-commit")
	}
}

func TestCollect_NoGitNoEnv(t *testing.T) {
	dir := t.TempDir() // not a git repo

	t.Setenv("GIT_COMMIT", "")
	t.Setenv("GITHUB_SHA", "")

	info := Collect(dir)

	if info.Commit != "" {
		t.Errorf("Commit = %q, want empty string", info.Commit)
	}
	if info.Branch != "" {
		t.Errorf("Branch = %q, want empty string", info.Branch)
	}
	if info.Dirty {
		t.Error("expected Dirty=false when git is unavailable")
	}
}

func TestCollect_GitRepoTakesPrecedenceOverEnv(t *testing.T) {
	dir := initRepo(t)
	sha := commitFile(t, dir, "f.txt", "content", "commit")

	t.Setenv("GIT_COMMIT", "env-value-should-be-ignored")

	info := Collect(dir)

	if info.Commit != sha {
		t.Errorf("Commit = %q, want %q (git should take precedence over env)", info.Commit, sha)
	}
}
