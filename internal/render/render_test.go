package render

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "update golden files")

var testCtx = TemplateCtx{
	GitCommit: "abc1234",
	GitBranch: "MAIN",
	GitDirty:  false,
	Timestamp: time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC),
	Initiator: "tester",
	Version:   "0.1.0-test",
	Env:       map[string]string{"APP_ENV": "test"},
}

func TestRender(t *testing.T) {
	t.Setenv("ARTEMIS_TEST_VAR", "hello-from-test")

	templatesDir := filepath.Join("testdata", "templates")
	goldenDir := filepath.Join("testdata", "golden")
	outputDir := t.TempDir()

	if err := Render(templatesDir, outputDir, testCtx); err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	err := filepath.Walk(goldenDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(goldenDir, path)
		if err != nil {
			return err
		}

		got, err := os.ReadFile(filepath.Join(outputDir, rel))
		if err != nil {
			t.Errorf("missing output file %s: %v", rel, err)
			return nil
		}

		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			return os.WriteFile(path, got, 0o644)
		}

		want, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reading golden file %s: %v", rel, err)
			return nil
		}

		if string(got) != string(want) {
			t.Errorf("output mismatch for %s:\ngot:\n%s\nwant:\n%s", rel, got, want)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walking golden dir: %v", err)
	}
}

func TestRender_StaticFileCopied(t *testing.T) {
	templatesDir := filepath.Join("testdata", "templates")
	outputDir := t.TempDir()

	if err := Render(templatesDir, outputDir, testCtx); err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "static.txt"))
	if err != nil {
		t.Fatalf("static file not copied: %v", err)
	}

	want := "I am a static file.\n"
	if string(content) != want {
		t.Errorf("static file content mismatch:\ngot:  %q\nwant: %q", string(content), want)
	}
}

func TestRender_PreservesFileMode(t *testing.T) {
	tmpTemplates := t.TempDir()
	outputDir := t.TempDir()

	// Create a template with executable mode.
	scriptPath := filepath.Join(tmpTemplates, "run.sh.tmpl")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho {{ .Version }}\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Render(tmpTemplates, outputDir, testCtx); err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	info, err := os.Stat(filepath.Join(outputDir, "run.sh"))
	if err != nil {
		t.Fatalf("output file missing: %v", err)
	}

	// Check that the executable bit is preserved.
	if info.Mode()&0o111 == 0 {
		t.Errorf("expected executable mode, got %v", info.Mode())
	}
}

func TestRender_NestedDirectories(t *testing.T) {
	templatesDir := filepath.Join("testdata", "templates")
	outputDir := t.TempDir()

	if err := Render(templatesDir, outputDir, testCtx); err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Verify nested output exists.
	content, err := os.ReadFile(filepath.Join(outputDir, "sub", "nested.txt"))
	if err != nil {
		t.Fatalf("nested file not rendered: %v", err)
	}

	if len(content) == 0 {
		t.Error("nested file is empty")
	}
}

func TestRender_BadTemplate(t *testing.T) {
	tmpTemplates := t.TempDir()
	outputDir := t.TempDir()

	// Write an invalid template.
	badPath := filepath.Join(tmpTemplates, "bad.txt.tmpl")
	if err := os.WriteFile(badPath, []byte("{{ .Missing | noSuchFunc }}"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Render(tmpTemplates, outputDir, testCtx)
	if err == nil {
		t.Fatal("expected error for bad template, got nil")
	}
}

func TestRender_RejectsSymlinkInTemplates(t *testing.T) {
	// A symlink in the templates tree must not be read through: doing so copies
	// a file from outside templatesDir into the output.
	secretDir := t.TempDir()
	secret := filepath.Join(secretDir, "secret.txt")
	if err := os.WriteFile(secret, []byte("TOP-SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}

	tmpTemplates := t.TempDir()
	outputDir := t.TempDir()
	if err := os.Symlink(secret, filepath.Join(tmpTemplates, "leak.txt")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	if err := Render(tmpTemplates, outputDir, testCtx); err == nil {
		t.Fatal("expected error for symlink in templates dir, got nil")
	}

	if got, err := os.ReadFile(filepath.Join(outputDir, "leak.txt")); err == nil {
		t.Errorf("symlink target was copied into output: %q", got)
	}
}

func TestRender_DoesNotWriteThroughOutputSymlink(t *testing.T) {
	// A symlink planted in the output dir must not redirect a rendered file to
	// its target.
	victimDir := t.TempDir()
	victim := filepath.Join(victimDir, "victim.txt")
	if err := os.WriteFile(victim, []byte("ORIGINAL"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpTemplates := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpTemplates, "x.txt.tmpl"), []byte("RENDERED"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	if err := os.Symlink(victim, filepath.Join(outputDir, "x.txt")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	if err := Render(tmpTemplates, outputDir, testCtx); err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ORIGINAL" {
		t.Errorf("file outside outputDir was overwritten: got %q, want %q", got, "ORIGINAL")
	}

	// The rendered content must land at the real path inside outputDir.
	got, err = os.ReadFile(filepath.Join(outputDir, "x.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "RENDERED" {
		t.Errorf("output content mismatch: got %q, want %q", got, "RENDERED")
	}
}

func TestRender_OverwritesExistingRegularFile(t *testing.T) {
	// createOutput unlinks before creating with O_EXCL; rendering twice into the
	// same directory must still succeed.
	tmpTemplates := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpTemplates, "out.txt.tmpl"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outputDir, "out.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := Render(tmpTemplates, outputDir, testCtx); err != nil {
			t.Fatalf("Render() run %d error: %v", i+1, err)
		}
	}

	got, err := os.ReadFile(filepath.Join(outputDir, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v2" {
		t.Errorf("existing file not replaced: got %q, want %q", got, "v2")
	}
}

func TestRender_MissingTemplatesDir(t *testing.T) {
	outputDir := t.TempDir()

	err := Render("/nonexistent/path", outputDir, testCtx)
	if err == nil {
		t.Fatal("expected error for missing templates dir, got nil")
	}
}
