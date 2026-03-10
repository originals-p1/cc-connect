package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ProjectInfo describes one switchable project discovered under the workspace root.
type ProjectInfo struct {
	Name    string
	Path    string
	GitRoot string
}

// ProjectCatalog is the set of projects discovered under one workspace root.
type ProjectCatalog struct {
	Root     string
	Projects map[string]ProjectInfo
}

// ScanWorkspace scans direct child directories under root and returns git-backed projects.
func ScanWorkspace(root string, requireGit bool) (*ProjectCatalog, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read workspace root: %w", err)
	}

	catalog := &ProjectCatalog{
		Root:     root,
		Projects: make(map[string]ProjectInfo),
	}

	// Sort entries to keep scan results deterministic.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		if !entry.IsDir() {
			info, err := os.Stat(path)
			if err != nil || !info.IsDir() {
				continue
			}
		}

		resolved := path
		if realPath, err := filepath.EvalSymlinks(path); err == nil {
			resolved = realPath
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			continue
		}

		gitRoot := resolved
		if requireGit {
			info, err := os.Stat(filepath.Join(resolved, ".git"))
			if err != nil || !info.IsDir() {
				continue
			}
		}

		name := filepath.Base(gitRoot)
		if _, exists := catalog.Projects[name]; exists {
			return nil, fmt.Errorf("duplicate project name %q found while scanning %s", name, root)
		}

		catalog.Projects[name] = ProjectInfo{
			Name:    name,
			Path:    resolved,
			GitRoot: gitRoot,
		}
	}

	return catalog, nil
}
