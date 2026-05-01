package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/azahorscak/artemis/internal/gitinfo"
)

var fixedTime = time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)

func TestNew_SchemaVersion(t *testing.T) {
	m := New("0.1.0", "assets/templates", "sha256:abc", "alice", gitinfo.Info{}, fixedTime)
	if m.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", m.SchemaVersion)
	}
}

func TestNew_Tool(t *testing.T) {
	m := New("1.2.3", "assets/templates", "sha256:abc", "alice", gitinfo.Info{}, fixedTime)
	if m.Tool.Name != "artemis" {
		t.Errorf("Tool.Name = %q, want %q", m.Tool.Name, "artemis")
	}
	if m.Tool.Version != "1.2.3" {
		t.Errorf("Tool.Version = %q, want %q", m.Tool.Version, "1.2.3")
	}
}

func TestNew_Source(t *testing.T) {
	m := New("0.1.0", "assets/templates", "sha256:deadbeef", "alice", gitinfo.Info{}, fixedTime)
	if m.Source.TemplatesDir != "assets/templates" {
		t.Errorf("Source.TemplatesDir = %q, want %q", m.Source.TemplatesDir, "assets/templates")
	}
	if m.Source.Hash != "sha256:deadbeef" {
		t.Errorf("Source.Hash = %q, want %q", m.Source.Hash, "sha256:deadbeef")
	}
}

func TestNew_Git(t *testing.T) {
	gi := gitinfo.Info{
		Commit: "abc123",
		Branch: "main",
		Dirty:  true,
	}
	m := New("0.1.0", "assets/templates", "sha256:abc", "alice", gi, fixedTime)
	if m.Git.Commit != "abc123" {
		t.Errorf("Git.Commit = %q, want %q", m.Git.Commit, "abc123")
	}
	if m.Git.Branch != "main" {
		t.Errorf("Git.Branch = %q, want %q", m.Git.Branch, "main")
	}
	if !m.Git.Dirty {
		t.Error("Git.Dirty = false, want true")
	}
}

func TestNew_Build(t *testing.T) {
	m := New("0.1.0", "assets/templates", "sha256:abc", "flox-build:alice", gitinfo.Info{}, fixedTime)

	if m.Build.Initiator != "flox-build:alice" {
		t.Errorf("Build.Initiator = %q, want %q", m.Build.Initiator, "flox-build:alice")
	}
	if m.Build.Timestamp != "2026-04-15T12:00:00Z" {
		t.Errorf("Build.Timestamp = %q, want %q", m.Build.Timestamp, "2026-04-15T12:00:00Z")
	}
}

func TestWriteFile_CreatesJSON(t *testing.T) {
	dir := t.TempDir()

	m := Metadata{
		SchemaVersion: 1,
		Tool:          Tool{Name: "artemis", Version: "0.1.0"},
		Source:        Source{TemplatesDir: "assets/templates", Hash: "sha256:abc"},
		Git:           Git{Commit: "abc123", Branch: "main", Dirty: false},
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
	if got.Git.Dirty {
		t.Error("Git.Dirty = true, want false")
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
	gi := gitinfo.Info{Commit: "abc", Branch: "main", Dirty: false}
	m := New("0.1.0", "assets/templates", "sha256:abc", "alice", gi, fixedTime)
	if m.Git.Dirty {
		t.Error("Git.Dirty = true, want false")
	}
}

func TestWriteFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	gi := gitinfo.Info{Commit: "deadbeef", Branch: "feature/x", Dirty: true}
	original := New("2.0.0", "my/templates", "sha256:cafebabe", "ci-bot", gi, fixedTime)

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

	// Compare all fields.
	if roundTripped.SchemaVersion != original.SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", roundTripped.SchemaVersion, original.SchemaVersion)
	}
	if roundTripped.Tool != original.Tool {
		t.Errorf("Tool = %+v, want %+v", roundTripped.Tool, original.Tool)
	}
	if roundTripped.Source != original.Source {
		t.Errorf("Source = %+v, want %+v", roundTripped.Source, original.Source)
	}
	if roundTripped.Git != original.Git {
		t.Errorf("Git = %+v, want %+v", roundTripped.Git, original.Git)
	}
	if roundTripped.Build != original.Build {
		t.Errorf("Build = %+v, want %+v", roundTripped.Build, original.Build)
	}
}
