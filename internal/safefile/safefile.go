// Package safefile creates output files without following planted symlinks.
package safefile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Create opens dst for writing, refusing to follow a pre-existing symlink.
// The parent directory must already exist.
//
// O_CREATE|O_TRUNC alone follows a symlink at dst and writes through it, so a
// link planted in the output directory redirects a written file anywhere the
// process can write. Unlinking first drops the link itself rather than its
// target, and O_EXCL then closes the gap: if anything reappears at dst before
// the open — symlink or not — the create fails instead of writing through it.
func Create(dst string, mode fs.FileMode) (*os.File, error) {
	if err := os.Remove(dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("clearing existing output path %s: %w", dst, err)
	}

	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return nil, fmt.Errorf("creating %s: %w", dst, err)
	}

	return f, nil
}
