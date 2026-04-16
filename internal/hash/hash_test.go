package hash

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a test helper that creates a file with the given content and mode.
func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func TestDir_Deterministic(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "a.txt"), "alpha", 0o644)
	writeFile(t, filepath.Join(dir, "b.txt"), "bravo", 0o644)
	writeFile(t, filepath.Join(dir, "sub", "c.txt"), "charlie", 0o644)

	h1, err := Dir(dir)
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}
	h2, err := Dir(dir)
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}

	if h1 != h2 {
		t.Errorf("same directory produced different hashes: %s vs %s", h1, h2)
	}
}

func TestDir_OrderIndependence(t *testing.T) {
	// Create two directories with the same files added in different order.
	// Because the hasher sorts by relpath, both must produce the same hash.

	dirA := t.TempDir()
	// Write in alphabetical order.
	writeFile(t, filepath.Join(dirA, "x.txt"), "xray", 0o644)
	writeFile(t, filepath.Join(dirA, "y.txt"), "yankee", 0o644)
	writeFile(t, filepath.Join(dirA, "z.txt"), "zulu", 0o644)

	dirB := t.TempDir()
	// Write in reverse order.
	writeFile(t, filepath.Join(dirB, "z.txt"), "zulu", 0o644)
	writeFile(t, filepath.Join(dirB, "y.txt"), "yankee", 0o644)
	writeFile(t, filepath.Join(dirB, "x.txt"), "xray", 0o644)

	hashA, err := Dir(dirA)
	if err != nil {
		t.Fatalf("hashing dirA: %v", err)
	}
	hashB, err := Dir(dirB)
	if err != nil {
		t.Fatalf("hashing dirB: %v", err)
	}

	if hashA != hashB {
		t.Errorf("order-independent hash failed: %s vs %s", hashA, hashB)
	}
}

func TestDir_ContentChange(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "file.txt"), "original", 0o644)
	h1, err := Dir(dir)
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}

	writeFile(t, filepath.Join(dir, "file.txt"), "modified", 0o644)
	h2, err := Dir(dir)
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}

	if h1 == h2 {
		t.Error("different content produced the same hash")
	}
}

func TestDir_ModeChange(t *testing.T) {
	dirA := t.TempDir()
	writeFile(t, filepath.Join(dirA, "script.sh"), "#!/bin/sh", 0o644)

	dirB := t.TempDir()
	writeFile(t, filepath.Join(dirB, "script.sh"), "#!/bin/sh", 0o755)

	hashA, err := Dir(dirA)
	if err != nil {
		t.Fatalf("hashing dirA: %v", err)
	}
	hashB, err := Dir(dirB)
	if err != nil {
		t.Fatalf("hashing dirB: %v", err)
	}

	if hashA == hashB {
		t.Error("different file modes produced the same hash")
	}
}

func TestDir_PrefixFormat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "f.txt"), "hello", 0o644)

	h, err := Dir(dir)
	if err != nil {
		t.Fatal(err)
	}

	const prefix = "sha256:"
	if len(h) <= len(prefix) || h[:len(prefix)] != prefix {
		t.Errorf("hash should start with %q, got %q", prefix, h)
	}

	// SHA-256 hex digest is 64 characters.
	hexPart := h[len(prefix):]
	if len(hexPart) != 64 {
		t.Errorf("expected 64 hex chars, got %d: %s", len(hexPart), hexPart)
	}
}

func TestDir_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	h, err := Dir(dir)
	if err != nil {
		t.Fatalf("hashing empty dir: %v", err)
	}

	if h == "" {
		t.Error("expected non-empty hash for empty directory")
	}
}

func TestDir_Subdirectories(t *testing.T) {
	// Files in subdirectories use forward-slash relative paths,
	// so the same layout in two temp dirs must hash identically.
	dirA := t.TempDir()
	writeFile(t, filepath.Join(dirA, "d1", "a.txt"), "one", 0o644)
	writeFile(t, filepath.Join(dirA, "d2", "b.txt"), "two", 0o644)

	dirB := t.TempDir()
	writeFile(t, filepath.Join(dirB, "d2", "b.txt"), "two", 0o644)
	writeFile(t, filepath.Join(dirB, "d1", "a.txt"), "one", 0o644)

	hashA, err := Dir(dirA)
	if err != nil {
		t.Fatalf("hashing dirA: %v", err)
	}
	hashB, err := Dir(dirB)
	if err != nil {
		t.Fatalf("hashing dirB: %v", err)
	}

	if hashA != hashB {
		t.Errorf("subdirectory order independence failed: %s vs %s", hashA, hashB)
	}
}
