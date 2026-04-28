// Package metadata defines the build metadata schema and writes metadata.json.
package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/azahorscak/artemis/internal/gitinfo"
)

// TODO(metadata): add outputs[]{path,sha256} with per-file hashes

// Metadata represents the top-level metadata.json structure.
// Increment SchemaVersion whenever fields are removed or semantics change in a breaking way.
type Metadata struct {
	SchemaVersion int    `json:"schemaVersion"` // currently 1; consumers should reject unknown versions
	Tool          Tool   `json:"tool"`
	Source        Source `json:"source"`
	Git           Git    `json:"git"`
	Build         Build  `json:"build"`
}

// Tool identifies the program that produced the output.
type Tool struct {
	Name    string `json:"name"`    // always "artemis"
	Version string `json:"version"` // value injected via ldflags, or "dev" for local builds
}

// Source describes the template inputs used to produce this output.
type Source struct {
	TemplatesDir string `json:"templatesDir"` // path passed to --templates-dir
	Hash         string `json:"hash"`         // "sha256:<hex>" digest of the entire templates directory
}

// Git captures the repository state at build time for traceability.
type Git struct {
	Commit string `json:"commit"` // full SHA-1, or empty when git is unavailable
	Branch string `json:"branch"` // branch name, or empty on detached HEAD / no git
	Dirty  bool   `json:"dirty"`  // true when uncommitted changes existed at build time
}

// Build holds information about when and by whom the output was produced.
type Build struct {
	Timestamp string `json:"timestamp"` // UTC build time in RFC 3339 format
	Initiator string `json:"initiator"` // resolved identity of the build invoker
}

// New creates a Metadata value from the provided inputs.
// The timestamp is set to the current UTC time in RFC 3339 format.
func New(version, templatesDir, hash, initiator string, gi gitinfo.Info) Metadata {
	return Metadata{
		SchemaVersion: 1,
		Tool: Tool{
			Name:    "artemis",
			Version: version,
		},
		Source: Source{
			TemplatesDir: templatesDir,
			Hash:         hash,
		},
		Git: Git{
			Commit: gi.Commit,
			Branch: gi.Branch,
			Dirty:  gi.Dirty,
		},
		Build: Build{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Initiator: initiator,
		},
	}
}

// WriteFile writes the metadata as indented JSON to outputDir/metadata.json.
// The file always ends with a trailing newline for POSIX compliance.
func WriteFile(outputDir string, m Metadata) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n') // POSIX: text files must end with a newline
	return os.WriteFile(filepath.Join(outputDir, "metadata.json"), data, 0o644)
}
