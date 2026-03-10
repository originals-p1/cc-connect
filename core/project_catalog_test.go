package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func TestScanWorkspaceRequireGit(t *testing.T) {
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	plain := filepath.Join(root, "notes")
	nested := filepath.Join(root, "nested", "repo-c")

	mkdirAll(t, filepath.Join(repoA, ".git"))
	mkdirAll(t, filepath.Join(repoB, ".git"))
	mkdirAll(t, plain)
	mkdirAll(t, filepath.Join(nested, ".git"))

	catalog, err := ScanWorkspace(root, true)
	if err != nil {
		t.Fatalf("ScanWorkspace() error = %v", err)
	}

	if len(catalog.Projects) != 2 {
		t.Fatalf("len(Projects) = %d, want 2", len(catalog.Projects))
	}
	if _, ok := catalog.Projects["repo-a"]; !ok {
		t.Fatalf("repo-a missing from catalog")
	}
	if _, ok := catalog.Projects["repo-b"]; !ok {
		t.Fatalf("repo-b missing from catalog")
	}
	if _, ok := catalog.Projects["notes"]; ok {
		t.Fatalf("non-git directory should not be included when require_git=true")
	}
	if _, ok := catalog.Projects["repo-c"]; ok {
		t.Fatalf("nested directory should not be included")
	}
}

func TestScanWorkspaceMissingRoot(t *testing.T) {
	_, err := ScanWorkspace(filepath.Join(t.TempDir(), "missing"), true)
	if err == nil {
		t.Fatal("ScanWorkspace() error = nil, want error")
	}
}

func TestScanWorkspaceDuplicateNames(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup for duplicate-name scan is not stable on Windows test env")
	}

	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	aliasRepoA := filepath.Join(root, "repo-a-alias")

	mkdirAll(t, filepath.Join(repoA, ".git"))
	if err := os.Symlink(repoA, aliasRepoA); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	_, err := ScanWorkspace(root, true)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("ScanWorkspace() error = %v, want duplicate-name error", err)
	}
}
