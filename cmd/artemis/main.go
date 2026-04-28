package main

import (
	"flag"
	"fmt"
	"os"
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

	// TODO(step7): wire up the execution pipeline using cfg.
	fmt.Fprintf(os.Stderr, "artemis %s: config parsed, pipeline not yet wired\n", cfg.Version)
	fmt.Fprintf(os.Stderr, "  templates-dir: %s\n", cfg.TemplatesDir)
	fmt.Fprintf(os.Stderr, "  output-dir:    %s\n", cfg.OutputDir)
	fmt.Fprintf(os.Stderr, "  initiator:     %s\n", cfg.Initiator)
	os.Exit(1)
}
