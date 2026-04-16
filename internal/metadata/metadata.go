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
type Metadata struct {
	SchemaVersion int     `json:"schemaVersion"`
	Tool          Tool    `json:"tool"`
	Source        Source  `json:"source"`
	Git           Git     `json:"git"`
	Build         Build   `json:"build"`
}

// Tool identifies the program that produced the output.
type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Source describes the template inputs.
type Source struct {
	TemplatesDir string `json:"templatesDir"`
	Hash         string `json:"hash"`
}

// Git captures the repository state at build time.
type Git struct {
	Commit string `json:"commit"`
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
}

// Build holds build-time information.
type Build struct {
	Timestamp string `json:"timestamp"`
	Initiator string `json:"initiator"`
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
func WriteFile(outputDir string, m Metadata) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(outputDir, "metadata.json"), data, 0o644)
}
