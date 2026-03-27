package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestExecuteSling_BlockedByDependency verifies that executeSling rejects beads
// whose direct dependencies are not closed.
func TestExecuteSling_BlockedByDependency(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	townRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(townRoot, ".beads"), 0o755); err != nil {
		t.Fatalf("failed to create .beads: %v", err)
	}

	binDir := filepath.Join(townRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir binDir: %v", err)
	}

	bdScript := `#!/bin/sh
case "$1" in
  show)
    echo '[{"title":"Blocked task","status":"open","assignee":"","description":"","dependencies":[{"id":"gt-dep1","title":"Blocker","status":"open"}]}]'
    ;;
esac
exit 0
`
	writeBDStub(t, binDir, bdScript, "")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	params := SlingParams{
		BeadID:   "gt-blocked1",
		RigName:  "testrig",
		TownRoot: townRoot,
	}

	result, err := executeSling(params)
	if err == nil {
		t.Fatal("expected error when slinging bead with open dependency, got nil")
	}
	if result.ErrMsg != "blocked" {
		t.Errorf("expected ErrMsg='blocked', got %q", result.ErrMsg)
	}
	if !strings.Contains(err.Error(), "gt-dep1") {
		t.Errorf("error should name the blocking dep: %v", err)
	}
	if !strings.Contains(err.Error(), "open") {
		t.Errorf("error should show dep status: %v", err)
	}
}

// TestExecuteSling_ForceBlocked bypasses the dependency check.
func TestExecuteSling_ForceBlocked(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	townRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(townRoot, ".beads"), 0o755); err != nil {
		t.Fatalf("failed to create .beads: %v", err)
	}

	binDir := filepath.Join(townRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir binDir: %v", err)
	}

	bdScript := `#!/bin/sh
case "$1" in
  show)
    echo '[{"title":"Blocked task","status":"open","assignee":"","description":"","dependencies":[{"id":"gt-dep2","title":"Blocker","status":"open"}]}]'
    ;;
esac
exit 0
`
	writeBDStub(t, binDir, bdScript, "")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	params := SlingParams{
		BeadID:       "gt-blocked2",
		RigName:      "testrig",
		TownRoot:     townRoot,
		ForceBlocked: true,
	}

	// Should NOT fail on dependency check — may fail later on polecat spawn (no tmux).
	_, err := executeSling(params)
	if err != nil && strings.Contains(err.Error(), "gt-dep2") {
		t.Errorf("--force-blocked should bypass dependency check, but got dep error: %v", err)
	}
}

// TestExecuteSling_ClosedDependencyAllowed verifies that beads with only closed
// dependencies can be dispatched.
func TestExecuteSling_ClosedDependencyAllowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	townRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(townRoot, ".beads"), 0o755); err != nil {
		t.Fatalf("failed to create .beads: %v", err)
	}

	binDir := filepath.Join(townRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir binDir: %v", err)
	}

	bdScript := `#!/bin/sh
case "$1" in
  show)
    echo '[{"title":"Ready task","status":"open","assignee":"","description":"","dependencies":[{"id":"gt-done1","title":"Done dep","status":"closed"}]}]'
    ;;
esac
exit 0
`
	writeBDStub(t, binDir, bdScript, "")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	params := SlingParams{
		BeadID:   "gt-ready1",
		RigName:  "testrig",
		TownRoot: townRoot,
	}

	_, err := executeSling(params)
	if err != nil && strings.Contains(err.Error(), "gt-done1") {
		t.Errorf("closed dependency should not block sling, but got dep error: %v", err)
	}
}

// TestExecuteSling_WispDepSkipped verifies that molecule wisps in the dependency
// list are ignored (they are internal scaffolding, not user-visible blockers).
func TestExecuteSling_WispDepSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	townRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(townRoot, ".beads"), 0o755); err != nil {
		t.Fatalf("failed to create .beads: %v", err)
	}

	binDir := filepath.Join(townRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir binDir: %v", err)
	}

	bdScript := `#!/bin/sh
case "$1" in
  show)
    echo '[{"title":"Wisp task","status":"open","assignee":"","description":"","dependencies":[{"id":"gt-wisp-abc123","title":"Formula step","status":"open"}]}]'
    ;;
esac
exit 0
`
	writeBDStub(t, binDir, bdScript, "")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	params := SlingParams{
		BeadID:   "gt-wisptest1",
		RigName:  "testrig",
		TownRoot: townRoot,
	}

	_, err := executeSling(params)
	if err != nil && strings.Contains(err.Error(), "gt-wisp-abc123") {
		t.Errorf("open wisp dep should not block sling, but got dep error: %v", err)
	}
}
