// Package hash computes a deterministic SHA-256 hash of a directory.
package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// entry holds the metadata for a single file used in the aggregate hash.
type entry struct {
	relPath string
	mode    fs.FileMode
	hash    string // hex-encoded SHA-256 of file contents
}

// Dir computes a deterministic SHA-256 hash over all files in dir.
// It walks the directory, collecting (relpath, mode, sha256(content)) tuples,
// sorts them by relative path, and hashes the sorted list to produce a single
// aggregate digest. The result is returned as "sha256:<hex>".
func Dir(dir string) (string, error) {
	var entries []entry

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		h, err := fileHash(path)
		if err != nil {
			return err
		}

		entries = append(entries, entry{
			relPath: filepath.ToSlash(rel),
			mode:    info.Mode(),
			hash:    h,
		})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking directory %s: %w", dir, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].relPath < entries[j].relPath
	})

	aggregate := sha256.New()
	for _, e := range entries {
		fmt.Fprintf(aggregate, "%s %o %s\n", e.relPath, e.mode, e.hash)
	}

	return "sha256:" + hex.EncodeToString(aggregate.Sum(nil)), nil
}

// fileHash returns the hex-encoded SHA-256 digest of a file's contents.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hashing %s: %w", path, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
