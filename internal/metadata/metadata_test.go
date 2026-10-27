package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/azahorscak/artemis/internal/gitinfo"
)

// testTime is a fixed build instant for tests that don't inspect the timestamp.
var testTime = time.Date(2026, 4, 15, 12, 34, 56, 0, time.UTC)

// boolPtr returns a pointer to v, for building tri-state Dirty values.
func boolPtr(v bool) *bool { return &v }

// sameDirty compares two tri-state Dirty values by value, not by pointer.
func sameDirty(a, b *bool) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func TestNew_SchemaVersion(t *testing.T) {
	m := New("0.1.0", "assets/templates", "sha256:abc", "alice", gitinfo.Info{}, testTime)
	if m.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", m.SchemaVersion)
	}
}

func TestNew_Tool(t *testing.T) {
	m := New("1.2.3", "assets/templates", "sha256:abc", "alice", gitinfo.Info{}, testTime)
	if m.Tool.Name != "artemis" {
		t.Errorf("Tool.Name = %q, want %q", m.Tool.Name, "artemis")
	}
	if m.Tool.Version != "1.2.3" {
		t.Errorf("Tool.Version = %q, want %q", m.Tool.Version, "1.2.3")
	}
}

func TestNew_Source(t *testing.T) {
	m := New("0.1.0", "assets/templates", "sha256:deadbeef", "alice", gitinfo.Info{}, testTime)
	if m.Source.TemplatesDir != "assets/templates" {
		t.Errorf("Source.TemplatesDir = %q, want %q", m.Source.TemplatesDir, "assets/templates")
	}
	if m.Source.Hash != "sha256:deadbeef" {
		t.Errorf("Source.Hash = %q, want %q", m.Source.Hash, "sha256:deadbeef")
	}
}

func TestNew_Git(t *testing.T) {
	gi := gitinfo.Info{
		Commit:       "abc123",
		CommitSource: gitinfo.SourceGit,
		Branch:       "main",
		Dirty:        boolPtr(true),
	}
	m := New("0.1.0", "assets/templates", "sha256:abc", "alice", gi, testTime)
	if m.Git.Commit != "abc123" {
		t.Errorf("Git.Commit = %q, want %q", m.Git.Commit, "abc123")
	}
	if m.Git.Branch != "main" {
		t.Errorf("Git.Branch = %q, want %q", m.Git.Branch, "main")
	}
	if m.Git.Dirty == nil || !*m.Git.Dirty {
		t.Error("Git.Dirty should be true")
	}
	if m.Git.CommitSource != gitinfo.SourceGit {
		t.Errorf("Git.CommitSource = %q, want %q", m.Git.CommitSource, gitinfo.SourceGit)
	}
}

func TestNew_Build(t *testing.T) {
	// A non-UTC input must still be recorded in UTC.
	ts := time.Date(2026, 4, 15, 14, 34, 56, 0, time.FixedZone("CEST", 2*60*60))
	m := New("0.1.0", "assets/templates", "sha256:abc", "flox-build:alice", gitinfo.Info{}, ts)

	if m.Build.Initiator != "flox-build:alice" {
		t.Errorf("Build.Initiator = %q, want %q", m.Build.Initiator, "flox-build:alice")
	}

	if want := "2026-04-15T12:34:56Z"; m.Build.Timestamp != want {
		t.Errorf("Build.Timestamp = %q, want %q", m.Build.Timestamp, want)
	}
}

func TestWriteFile_CreatesJSON(t *testing.T) {
	dir := t.TempDir()

	m := Metadata{
		SchemaVersion: 1,
		Tool:          Tool{Name: "artemis", Version: "0.1.0"},
		Source:        Source{TemplatesDir: "assets/templates", Hash: "sha256:abc"},
		Git:           Git{Commit: "abc123", CommitSource: gitinfo.SourceGit, Branch: "main", Dirty: boolPtr(false)},
		Build:         Build{Timestamp: "2026-04-15T12:34:56Z", Initiator: "alice"},
	}

	if err := WriteFile(dir, m); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatalf("reading metadata.json: %v", err)
	}

	var got Metadata
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshalling metadata.json: %v", err)
	}

	if got.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", got.SchemaVersion)
	}
	if got.Tool.Name != "artemis" {
		t.Errorf("Tool.Name = %q, want %q", got.Tool.Name, "artemis")
	}
	if got.Tool.Version != "0.1.0" {
		t.Errorf("Tool.Version = %q, want %q", got.Tool.Version, "0.1.0")
	}
	if got.Source.Hash != "sha256:abc" {
		t.Errorf("Source.Hash = %q, want %q", got.Source.Hash, "sha256:abc")
	}
	if got.Git.Commit != "abc123" {
		t.Errorf("Git.Commit = %q, want %q", got.Git.Commit, "abc123")
	}
	if got.Git.Branch != "main" {
		t.Errorf("Git.Branch = %q, want %q", got.Git.Branch, "main")
	}
	if got.Git.Dirty == nil || *got.Git.Dirty {
		t.Error("Git.Dirty should round-trip as false")
	}
	if got.Build.Timestamp != "2026-04-15T12:34:56Z" {
		t.Errorf("Build.Timestamp = %q, want %q", got.Build.Timestamp, "2026-04-15T12:34:56Z")
	}
	if got.Build.Initiator != "alice" {
		t.Errorf("Build.Initiator = %q, want %q", got.Build.Initiator, "alice")
	}
}

func TestWriteFile_IndentedJSON(t *testing.T) {
	dir := t.TempDir()

	m := Metadata{
		SchemaVersion: 1,
		Tool:          Tool{Name: "artemis", Version: "0.1.0"},
	}

	if err := WriteFile(dir, m); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatalf("reading metadata.json: %v", err)
	}

	// Verify the output is indented (contains newlines and spaces, not compact).
	raw := string(data)
	expected, _ := json.MarshalIndent(m, "", "  ")
	expected = append(expected, '\n')
	if raw != string(expected) {
		t.Errorf("WriteFile output is not properly indented:\ngot:\n%s\nwant:\n%s", raw, expected)
	}
}

func TestWriteFile_TrailingNewline(t *testing.T) {
	dir := t.TempDir()

	m := Metadata{SchemaVersion: 1}

	if err := WriteFile(dir, m); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatalf("reading metadata.json: %v", err)
	}

	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Error("metadata.json does not end with a trailing newline")
	}
}

func TestWriteFile_ErrorOnBadDir(t *testing.T) {
	err := WriteFile("/nonexistent/path", Metadata{})
	if err == nil {
		t.Error("expected error writing to nonexistent directory, got nil")
	}
}

func TestNew_GitDirtyFalse(t *testing.T) {
	gi := gitinfo.Info{Commit: "abc", Branch: "main", Dirty: boolPtr(false)}
	m := New("0.1.0", "assets/templates", "sha256:abc", "alice", gi, testTime)
	if m.Git.Dirty == nil || *m.Git.Dirty {
		t.Error("Git.Dirty should be false")
	}
}

func TestWriteFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	gi := gitinfo.Info{Commit: "deadbeef", CommitSource: gitinfo.SourceGit, Branch: "feature/x", Dirty: boolPtr(true)}
	original := New("2.0.0", "my/templates", "sha256:cafebabe", "ci-bot", gi, testTime)

	if err := WriteFile(dir, original); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatalf("reading metadata.json: %v", err)
	}

	var roundTripped Metadata
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshalling metadata.json: %v", err)
	}

	// Compare all fields except timestamp (tested separately).
	if roundTripped.SchemaVersion != original.SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", roundTripped.SchemaVersion, original.SchemaVersion)
	}
	if roundTripped.Tool != original.Tool {
		t.Errorf("Tool = %+v, want %+v", roundTripped.Tool, original.Tool)
	}
	if roundTripped.Source != original.Source {
		t.Errorf("Source = %+v, want %+v", roundTripped.Source, original.Source)
	}
	if roundTripped.Git.Commit != original.Git.Commit ||
		roundTripped.Git.CommitSource != original.Git.CommitSource ||
		roundTripped.Git.Branch != original.Git.Branch ||
		!sameDirty(roundTripped.Git.Dirty, original.Git.Dirty) {
		t.Errorf("Git = %+v, want %+v", roundTripped.Git, original.Git)
	}
	if roundTripped.Build != original.Build {
		t.Errorf("Build = %+v, want %+v", roundTripped.Build, original.Build)
	}
}

func TestWriteFile_DoesNotWriteThroughSymlink(t *testing.T) {
	// A symlink planted at metadata.json must not redirect the write to its
	// target, matching render's handling of the other output files.
	victim := filepath.Join(t.TempDir(), "victim.txt")
	if err := os.WriteFile(victim, []byte("ORIGINAL"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.Symlink(victim, filepath.Join(dir, "metadata.json")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	m := New("0.1.0", "assets/templates", "sha256:abc", "alice", gitinfo.Info{}, testTime)
	if err := WriteFile(dir, m); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ORIGINAL" {
		t.Errorf("file outside dir was overwritten: got %q, want %q", got, "ORIGINAL")
	}

	fi, err := os.Lstat(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !fi.Mode().IsRegular() {
		t.Errorf("metadata.json mode = %v, want regular file", fi.Mode())
	}
}
