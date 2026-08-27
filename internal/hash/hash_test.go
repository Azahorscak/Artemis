package hash

import (
	"crypto/sha256"
	"encoding/hex"
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

func TestDir_RejectsSymlink(t *testing.T) {
	// Hashing must not read through a symlink: the digest would then cover a
	// file outside dir, and would change when that unrelated file changed.
	secretDir := t.TempDir()
	secret := filepath.Join(secretDir, "secret.txt")
	writeFile(t, secret, "TOP-SECRET", 0o600)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "real.txt"), "content", 0o644)
	if err := os.Symlink(secret, filepath.Join(dir, "link.txt")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	if _, err := Dir(dir); err == nil {
		t.Fatal("expected error for symlink in hashed dir, got nil")
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

func TestDir_PathFramingIsNotForgeable(t *testing.T) {
	// A file whose *name* embeds what looks like a complete second record must
	// not hash the same as the two ordinary files that record describes.
	// Before the length prefix these two trees collided.
	emptySHA := hex.EncodeToString(sha256.New().Sum(nil))

	forged := t.TempDir()
	name := "a 644 " + emptySHA + "\nb"
	if err := os.WriteFile(filepath.Join(forged, name), []byte("HELLO"), 0o644); err != nil {
		t.Skipf("filesystem rejected adversarial filename: %v", err)
	}

	ordinary := t.TempDir()
	writeFile(t, filepath.Join(ordinary, "a"), "", 0o644)
	writeFile(t, filepath.Join(ordinary, "b"), "HELLO", 0o644)

	forgedHash, err := Dir(forged)
	if err != nil {
		t.Fatalf("hashing forged tree: %v", err)
	}
	ordinaryHash, err := Dir(ordinary)
	if err != nil {
		t.Fatalf("hashing ordinary tree: %v", err)
	}

	if forgedHash == ordinaryHash {
		t.Errorf("distinct trees collided: both hashed to %s", forgedHash)
	}
}

func TestDir_PathBoundaryIsUnambiguous(t *testing.T) {
	// Two trees whose concatenated path bytes are identical but split
	// differently. Only the length prefix keeps them apart.
	a := t.TempDir()
	writeFile(t, filepath.Join(a, "ab"), "x", 0o644)
	writeFile(t, filepath.Join(a, "c"), "x", 0o644)

	b := t.TempDir()
	writeFile(t, filepath.Join(b, "a"), "x", 0o644)
	writeFile(t, filepath.Join(b, "bc"), "x", 0o644)

	hashA, err := Dir(a)
	if err != nil {
		t.Fatal(err)
	}
	hashB, err := Dir(b)
	if err != nil {
		t.Fatal(err)
	}

	if hashA == hashB {
		t.Errorf("different path splits collided: both hashed to %s", hashA)
	}
}
