package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NichiyaOba/pokemon-dot-daze/progress"
	"github.com/NichiyaOba/pokemon-dot-daze/sprite"
)

func TestParseFlagsDefaults(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY_OWNER", "")
	t.Setenv("GITHUB_TOKEN", "")

	cfg, err := parseFlags(nil)
	if err != nil {
		t.Fatalf("parseFlags(nil) returned %v", err)
	}
	if cfg.dex != 1 {
		t.Errorf("dex = %d, want 1", cfg.dex)
	}
	if cfg.commitsSet {
		t.Error("commitsSet = true, want false when --commits was not passed")
	}
	if want := filepath.Join("assets", "pokemon.svg"); cfg.out != want {
		t.Errorf("out = %q, want %q", cfg.out, want)
	}
	if !cfg.caption {
		t.Error("caption = false, want true by default")
	}
}

func TestParseFlagsReadsEnvironment(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY_OWNER", "octocat")
	t.Setenv("GITHUB_TOKEN", "secret")

	cfg, err := parseFlags(nil)
	if err != nil {
		t.Fatalf("parseFlags(nil) returned %v", err)
	}
	if cfg.user != "octocat" {
		t.Errorf("user = %q, want octocat", cfg.user)
	}
	if cfg.token != "secret" {
		t.Errorf("token = %q, want secret", cfg.token)
	}
}

func TestParseFlagsRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"negative commits", []string{"--commits", "-5"}},
		{"minus one commits", []string{"--commits", "-1"}},
		{"empty out path", []string{"--out", ""}},
		{"unknown flag", []string{"--nope"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseFlags(tt.args); err == nil {
				t.Error("parseFlags() = nil error, want an error")
			}
		})
	}
}

// Zero must be usable: a brand-new account has no contributions yet.
func TestZeroCommitsIsDistinctFromUnset(t *testing.T) {
	cfg, err := parseFlags([]string{"--commits", "0"})
	if err != nil {
		t.Fatalf("parseFlags returned %v", err)
	}
	if !cfg.commitsSet {
		t.Fatal("commitsSet = false, want true when --commits was passed")
	}
	if cfg.commits != 0 {
		t.Fatalf("commits = %d, want 0", cfg.commits)
	}

	got, err := resolveCommits(context.Background(), cfg)
	if err != nil {
		t.Fatalf("resolveCommits returned %v", err)
	}
	if got != 0 {
		t.Errorf("resolveCommits() = %d, want 0 (must not fall through to the API)", got)
	}
}

func TestResolveCommitsPrefersExplicitValue(t *testing.T) {
	cfg := config{commits: 777, commitsSet: true}
	got, err := resolveCommits(context.Background(), cfg)
	if err != nil {
		t.Fatalf("resolveCommits returned %v", err)
	}
	if got != 777 {
		t.Errorf("resolveCommits() = %d, want 777", got)
	}
}

func TestResolveCommitsFailsWithoutCredentials(t *testing.T) {
	cfg := config{} // --commits not passed, so the API is consulted
	if _, err := resolveCommits(context.Background(), cfg); err == nil {
		t.Error("resolveCommits() = nil error, want an error when no token is set")
	}
}

// -h prints usage and exits successfully; it is not an error.
func TestHelpIsNotAnError(t *testing.T) {
	_, err := parseFlags([]string{"-h"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("parseFlags(-h) = %v, want flag.ErrHelp", err)
	}
}

func TestRenderOptionsRejectsBadTheme(t *testing.T) {
	cfg := config{cell: 16, gap: 0, theme: "neon"}
	if _, err := renderOptions(cfg); err == nil {
		t.Error("renderOptions() = nil error, want an error for an unknown theme")
	}
}

func TestRunWritesSVG(t *testing.T) {
	out := filepath.Join(t.TempDir(), "nested", "pokemon.svg")

	if err := run([]string{"--commits", "1000", "--out", out}); err != nil {
		t.Fatalf("run() returned %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "<svg") {
		t.Error("output does not start with an <svg> element")
	}
	if !strings.Contains(content, "Bulbasaur") {
		t.Error("output does not mention Bulbasaur")
	}
}

func TestRunIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.svg")
	b := filepath.Join(dir, "b.svg")

	for _, path := range []string{a, b} {
		if err := run([]string{"--commits", "1234", "--out", path}); err != nil {
			t.Fatalf("run() returned %v", err)
		}
	}

	first, err := os.ReadFile(a)
	if err != nil {
		t.Fatalf("reading %s: %v", a, err)
	}
	second, err := os.ReadFile(b)
	if err != nil {
		t.Fatalf("reading %s: %v", b, err)
	}
	if string(first) != string(second) {
		t.Error("two runs with the same input produced different files")
	}
}

func TestRunRejectsUnknownDex(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.svg")
	err := run([]string{"--commits", "10", "--dex", "999", "--out", out})
	if err == nil {
		t.Fatal("run() = nil error, want an error for an unregistered Pokédex number")
	}
	if !strings.Contains(err.Error(), "999") {
		t.Errorf("error %q does not mention the requested number", err)
	}
}

func TestRunRejectsInvalidRenderOptions(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.svg")
	if err := run([]string{"--commits", "10", "--cell", "0", "--out", out}); err == nil {
		t.Error("run() = nil error, want an error for a zero cell size")
	}
}

func TestReportProgressCoversBothStates(t *testing.T) {
	s, err := sprite.ByDex(1)
	if err != nil {
		t.Fatalf("ByDex: %v", err)
	}

	tests := []struct {
		name    string
		commits int
	}{
		{"in progress", 1000},
		{"complete", s.DotCount() * progress.CommitsPerDot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := progress.Compute(s, tt.commits)
			if err != nil {
				t.Fatalf("Compute: %v", err)
			}
			// Exercises both branches; output goes to stdout and is not asserted.
			reportProgress(p, "out.svg")
		})
	}
}

func TestWriteFileReportsUnwritablePaths(t *testing.T) {
	dir := t.TempDir()

	// A file where a directory is required.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("preparing the blocker file: %v", err)
	}
	if err := writeFile(filepath.Join(blocker, "nested", "out.svg"), []byte("x")); err == nil {
		t.Error("writeFile() = nil error, want an error when a parent path is a file")
	}

	// A read-only directory.
	readonly := filepath.Join(dir, "readonly")
	if err := os.Mkdir(readonly, 0o500); err != nil {
		t.Fatalf("preparing the read-only directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(readonly, 0o700) })

	if err := writeFile(filepath.Join(readonly, "out.svg"), []byte("x")); err == nil {
		t.Error("writeFile() = nil error, want an error for a read-only directory")
	}
}

func TestWriteFileAcceptsABarePath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := writeFile("bare.svg", []byte("x")); err != nil {
		t.Fatalf("writeFile() returned %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "bare.svg")); err != nil {
		t.Errorf("bare.svg was not created: %v", err)
	}
}

func TestRunRejectsAnUnwritableOutputPath(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("preparing the blocker file: %v", err)
	}
	if err := run([]string{"--commits", "10", "--out", filepath.Join(blocker, "out.svg")}); err == nil {
		t.Error("run() = nil error, want an error for an unwritable output path")
	}
}
