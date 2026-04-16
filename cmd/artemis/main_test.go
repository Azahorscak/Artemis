package main

import (
	"os"
	"testing"
)

func TestResolveInitiator_FlagTakesPrecedence(t *testing.T) {
	t.Setenv("FLOX_BUILD_INITIATOR", "env-value")
	got := resolveInitiator("flag-value")
	if got != "flag-value" {
		t.Errorf("resolveInitiator(\"flag-value\") = %q, want \"flag-value\"", got)
	}
}

func TestResolveInitiator_EnvFallbackOrder(t *testing.T) {
	// Clear all fallback vars.
	for _, env := range []string{"FLOX_BUILD_INITIATOR", "GITHUB_ACTOR", "CI_JOB_NAME", "USER"} {
		t.Setenv(env, "")
	}

	// Set only CI_JOB_NAME — should skip the first two empty vars.
	t.Setenv("CI_JOB_NAME", "ci-job")
	got := resolveInitiator("")
	if got != "ci-job" {
		t.Errorf("resolveInitiator(\"\") = %q, want \"ci-job\"", got)
	}
}

func TestResolveInitiator_FloxBuildFirst(t *testing.T) {
	t.Setenv("FLOX_BUILD_INITIATOR", "flox-user")
	t.Setenv("GITHUB_ACTOR", "gh-user")
	t.Setenv("USER", "local-user")

	got := resolveInitiator("")
	if got != "flox-user" {
		t.Errorf("resolveInitiator(\"\") = %q, want \"flox-user\"", got)
	}
}

func TestResolveInitiator_FallsBackToUnknown(t *testing.T) {
	for _, env := range []string{"FLOX_BUILD_INITIATOR", "GITHUB_ACTOR", "CI_JOB_NAME", "USER"} {
		t.Setenv(env, "")
	}
	got := resolveInitiator("")
	if got != "unknown" {
		t.Errorf("resolveInitiator(\"\") = %q, want \"unknown\"", got)
	}
}

func TestDefaultOutputDir_UsesOutEnv(t *testing.T) {
	t.Setenv("out", "/nix/store/abc123")
	got := defaultOutputDir()
	if got != "/nix/store/abc123" {
		t.Errorf("defaultOutputDir() = %q, want \"/nix/store/abc123\"", got)
	}
}

func TestDefaultOutputDir_FallsToBuild(t *testing.T) {
	// Unset $out explicitly.
	os.Unsetenv("out")
	t.Setenv("out", "")
	got := defaultOutputDir()
	if got != "./build" {
		t.Errorf("defaultOutputDir() = %q, want \"./build\"", got)
	}
}
