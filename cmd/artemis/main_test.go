package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveInitiator_FlagTakesPrecedence(t *testing.T) {
	t.Setenv("FLOX_BUILD_INITIATOR", "env-value")
	got := resolveInitiator("flag-value")
	if got != "flag-value" {
		t.Errorf("resolveInitiator(\"flag-value\") = %q, want \"flag-value\"", got)
	}
}

func TestResolveInitiator_EnvFallbackOrder(t *testing.T) {
	// Clear all fallback vars.
	for _, env := range []string{"FLOX_BUILD_INITIATOR", "GITHUB_ACTOR", "CI_JOB_NAME", "USER"} {
		t.Setenv(env, "")
	}

	// Set only CI_JOB_NAME — should skip the first two empty vars.
	t.Setenv("CI_JOB_NAME", "ci-job")
	got := resolveInitiator("")
	if got != "ci-job" {
		t.Errorf("resolveInitiator(\"\") = %q, want \"ci-job\"", got)
	}
}

func TestResolveInitiator_FloxBuildFirst(t *testing.T) {
	t.Setenv("FLOX_BUILD_INITIATOR", "flox-user")
	t.Setenv("GITHUB_ACTOR", "gh-user")
	t.Setenv("USER", "local-user")

	got := resolveInitiator("")
	if got != "flox-user" {
		t.Errorf("resolveInitiator(\"\") = %q, want \"flox-user\"", got)
	}
}

func TestResolveInitiator_FallsBackToUnknown(t *testing.T) {
	for _, env := range []string{"FLOX_BUILD_INITIATOR", "GITHUB_ACTOR", "CI_JOB_NAME", "USER"} {
		t.Setenv(env, "")
	}
	got := resolveInitiator("")
	if got != "unknown" {
		t.Errorf("resolveInitiator(\"\") = %q, want \"unknown\"", got)
	}
}

func TestDefaultOutputDir_UsesOutEnv(t *testing.T) {
	t.Setenv("out", "/nix/store/abc123")
	got := defaultOutputDir()
	if got != "/nix/store/abc123" {
		t.Errorf("defaultOutputDir() = %q, want \"/nix/store/abc123\"", got)
	}
}

func TestDefaultOutputDir_FallsToBuild(t *testing.T) {
	// Unset $out explicitly.
	os.Unsetenv("out")
	t.Setenv("out", "")
	got := defaultOutputDir()
	if got != "./build" {
		t.Errorf("defaultOutputDir() = %q, want \"./build\"", got)
	}
}

func TestRun_MissingTemplatesDir(t *testing.T) {
	cfg := Config{
		TemplatesDir: filepath.Join(t.TempDir(), "nonexistent"),
		OutputDir:    t.TempDir(),
		Initiator:    "test",
		Version:      "dev",
	}
	if err := run(cfg); err == nil {
		t.Fatal("expected error for missing templates dir, got nil")
	}
}

func TestRun_CreatesOutputDir(t *testing.T) {
	tmplDir := t.TempDir()
	// output dir does not exist yet
	outDir := filepath.Join(t.TempDir(), "created-by-run")

	cfg := Config{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
		Initiator:    "test",
		Version:      "dev",
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat(outDir); err != nil {
		t.Errorf("output dir not created: %v", err)
	}
}

func TestRun_RendersTemplate(t *testing.T) {
	tmplDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmplDir, "hello.txt.tmpl"), []byte("v{{ .Version }}"), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	cfg := Config{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
		Initiator:    "test",
		Version:      "1.2.3",
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(outDir, "hello.txt"))
	if err != nil {
		t.Fatalf("reading rendered output: %v", err)
	}
	if string(got) != "v1.2.3" {
		t.Errorf("rendered content = %q, want %q", string(got), "v1.2.3")
	}
}

func TestRun_CopiesStaticFile(t *testing.T) {
	tmplDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmplDir, "static.txt"), []byte("no template here"), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	cfg := Config{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
		Initiator:    "test",
		Version:      "dev",
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(outDir, "static.txt"))
	if err != nil {
		t.Fatalf("reading static output: %v", err)
	}
	if string(got) != "no template here" {
		t.Errorf("static content = %q, want %q", string(got), "no template here")
	}
}

func TestRun_WritesMetadataJSON(t *testing.T) {
	tmplDir := t.TempDir()
	outDir := t.TempDir()
	cfg := Config{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
		Initiator:    "ci-bot",
		Version:      "0.1.0",
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "metadata.json"))
	if err != nil {
		t.Fatalf("reading metadata.json: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal metadata.json: %v", err)
	}

	if m["schemaVersion"] != float64(1) {
		t.Errorf("schemaVersion = %v, want 1", m["schemaVersion"])
	}
	tool := m["tool"].(map[string]any)
	if tool["name"] != "artemis" {
		t.Errorf("tool.name = %v, want artemis", tool["name"])
	}
	if tool["version"] != "0.1.0" {
		t.Errorf("tool.version = %v, want 0.1.0", tool["version"])
	}
	build := m["build"].(map[string]any)
	if build["initiator"] != "ci-bot" {
		t.Errorf("build.initiator = %v, want ci-bot", build["initiator"])
	}
	source := m["source"].(map[string]any)
	if h, ok := source["hash"].(string); !ok || !strings.HasPrefix(h, "sha256:") {
		t.Errorf("source.hash = %v, want sha256:... prefix", source["hash"])
	}
}

func TestRun_MetadataNotIncludedInSourceHash(t *testing.T) {
	// Run twice on the same templates dir; source hash must be identical
	// regardless of metadata.json content (different timestamps).
	tmplDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmplDir, "f.txt.tmpl"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	readHash := func() string {
		outDir := t.TempDir()
		cfg := Config{TemplatesDir: tmplDir, OutputDir: outDir, Initiator: "x", Version: "dev"}
		if err := run(cfg); err != nil {
			t.Fatalf("run: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(outDir, "metadata.json"))
		var m map[string]any
		json.Unmarshal(data, &m)
		return m["source"].(map[string]any)["hash"].(string)
	}

	h1, h2 := readHash(), readHash()
	if h1 != h2 {
		t.Errorf("source hash not deterministic: %q vs %q", h1, h2)
	}
}

func TestRun_FailFastOnBadTemplate(t *testing.T) {
	tmplDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmplDir, "bad.txt.tmpl"), []byte("{{ .Undefined | badFunc }}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		TemplatesDir: tmplDir,
		OutputDir:    t.TempDir(),
		Initiator:    "test",
		Version:      "dev",
	}
	if err := run(cfg); err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
}
