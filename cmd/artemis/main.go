package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/azahorscak/artemis/internal/gitinfo"
	"github.com/azahorscak/artemis/internal/hash"
	"github.com/azahorscak/artemis/internal/metadata"
	"github.com/azahorscak/artemis/internal/render"
)

// version is set via ldflags at build time.
var version = "dev"

// Config holds the parsed CLI configuration the pipeline will consume.
type Config struct {
	TemplatesDir string
	OutputDir    string
	Initiator    string
	Version      string
}

// resolveInitiator returns the first non-empty value from the --initiator flag,
// a chain of environment variables, or "unknown" as a last resort.
func resolveInitiator(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	for _, env := range []string{
		"FLOX_BUILD_INITIATOR",
		"GITHUB_ACTOR",
		"CI_JOB_NAME",
		"USER",
	} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return "unknown"
}

// defaultOutputDir returns $out if set, otherwise "./build".
func defaultOutputDir() string {
	if v := os.Getenv("out"); v != "" {
		return v
	}
	return "./build"
}

// run executes the full pipeline: validate → hash → gitinfo → render → metadata.
func run(cfg Config) error {
	// 1. Validate inputs.
	if _, err := os.Stat(cfg.TemplatesDir); err != nil {
		return fmt.Errorf("templates dir: %w", err)
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	// 2. Hash templates dir (source hash, computed on input before rendering).
	sourceHash, err := hash.Dir(cfg.TemplatesDir)
	if err != nil {
		return fmt.Errorf("hashing templates: %w", err)
	}

	// 3. Collect git info so templates can reference commit/branch/dirty.
	gi := gitinfo.Collect(".")

	// 4. Render templates into output dir.
	ctx := render.TemplateCtx{
		GitCommit: gi.Commit,
		GitBranch: gi.Branch,
		GitDirty:  gi.Dirty,
		Timestamp: time.Now().UTC(),
		Initiator: cfg.Initiator,
		Version:   cfg.Version,
		Env:       nil,
	}
	if err := render.Render(cfg.TemplatesDir, cfg.OutputDir, ctx); err != nil {
		return fmt.Errorf("rendering templates: %w", err)
	}

	// 5. Write metadata.json last so it is excluded from the source hash.
	meta := metadata.New(cfg.Version, cfg.TemplatesDir, sourceHash, cfg.Initiator, gi)
	if err := metadata.WriteFile(cfg.OutputDir, meta); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}

	return nil
}

func main() {
	templatesDir := flag.String("templates-dir", "./assets/templates", "path to the templates directory")
	outputDir := flag.String("output-dir", defaultOutputDir(), "path to the output directory")
	initiator := flag.String("initiator", "", "build initiator identity (default: env lookup)")
	showVersion := flag.Bool("version", false, "print version and exit")

	flag.Parse()

	if *showVersion {
		fmt.Printf("artemis %s\n", version)
		return
	}

	cfg := Config{
		TemplatesDir: *templatesDir,
		OutputDir:    *outputDir,
		Initiator:    resolveInitiator(*initiator),
		Version:      version,
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "artemis: %v\n", err)
		os.Exit(1)
	}
}
