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
	Env:       map[string]string{"APP_ENV": "test", "ARTEMIS_TEST_VAR": "hello-from-test"},
}

func TestRender(t *testing.T) {
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

func TestRender_MissingTemplatesDir(t *testing.T) {
	outputDir := t.TempDir()

	err := Render("/nonexistent/path", outputDir, testCtx)
	if err == nil {
		t.Fatal("expected error for missing templates dir, got nil")
	}
}
