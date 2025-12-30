package filebrowser

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileNode represents a file or directory in the tree.
type FileNode struct {
	Name       string
	Path       string // Relative path from root
	IsDir      bool
	IsExpanded bool
	IsIgnored  bool // Set by gitignore
	Children   []*FileNode
	Parent     *FileNode
	Depth      int
	Size       int64
	ModTime    time.Time
}

// FileTree manages the hierarchical file structure.
type FileTree struct {
	Root      *FileNode
	RootDir   string
	FlatList  []*FileNode // Flattened visible nodes for cursor navigation
	gitIgnore *GitIgnore
}

// NewFileTree creates a new file tree rooted at the given directory.
func NewFileTree(rootDir string) *FileTree {
	return &FileTree{
		RootDir:   rootDir,
		FlatList:  make([]*FileNode, 0),
		gitIgnore: NewGitIgnore(),
	}
}

// Build initializes the tree by loading the root directory's children.
func (t *FileTree) Build() error {
	// Load .gitignore from root
	t.gitIgnore = NewGitIgnore()
	_ = t.gitIgnore.LoadFile(filepath.Join(t.RootDir, ".gitignore"))

	t.Root = &FileNode{
		Name:       filepath.Base(t.RootDir),
		Path:       "",
		IsDir:      true,
		IsExpanded: true,
		Depth:      -1, // Root is hidden, children start at depth 0
	}

	if err := t.loadChildren(t.Root); err != nil {
		return err
	}

	t.Flatten()
	return nil
}

// isSystemFile returns true for OS-generated files that clutter file browsers.
func isSystemFile(name string) bool {
	// Exact matches
	switch name {
	case ".DS_Store", ".Spotlight-V100", ".Trashes", ".fseventsd",
		".TemporaryItems", ".DocumentRevisions-V100",
		"Thumbs.db", "desktop.ini", "$RECYCLE.BIN":
		return true
	}
	// macOS resource fork files (._*)
	if strings.HasPrefix(name, "._") {
		return true
	}
	return false
}

// loadChildren populates a node's children from the filesystem.
func (t *FileTree) loadChildren(node *FileNode) error {
	fullPath := filepath.Join(t.RootDir, node.Path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return err
	}

	node.Children = make([]*FileNode, 0, len(entries))

	for _, entry := range entries {
		if isSystemFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		childPath := filepath.Join(node.Path, entry.Name())
		child := &FileNode{
			Name:      entry.Name(),
			Path:      childPath,
			IsDir:     entry.IsDir(),
			IsIgnored: t.gitIgnore.IsIgnored(childPath, entry.IsDir()),
			Parent:    node,
			Depth:     node.Depth + 1,
			Size:      info.Size(),
			ModTime:   info.ModTime(),
		}

		node.Children = append(node.Children, child)
	}

	sortChildren(node.Children)
	return nil
}

// sortChildren sorts nodes: directories first, then alphabetically by name.
func sortChildren(children []*FileNode) {
	sort.Slice(children, func(i, j int) bool {
		// Directories come before files
		if children[i].IsDir != children[j].IsDir {
			return children[i].IsDir
		}
		// Alphabetical, case-insensitive
		return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
	})
}

// Expand opens a directory node, loading children if needed.
func (t *FileTree) Expand(node *FileNode) error {
	if !node.IsDir {
		return nil
	}

	if len(node.Children) == 0 {
		if err := t.loadChildren(node); err != nil {
			return err
		}
	}

	node.IsExpanded = true
	t.Flatten()
	return nil
}

// Collapse closes a directory node.
func (t *FileTree) Collapse(node *FileNode) {
	node.IsExpanded = false
	t.Flatten()
}

// Toggle expands or collapses a directory node.
func (t *FileTree) Toggle(node *FileNode) error {
	if !node.IsDir {
		return nil
	}

	if node.IsExpanded {
		t.Collapse(node)
		return nil
	}
	return t.Expand(node)
}

// Flatten rebuilds the FlatList from visible nodes.
func (t *FileTree) Flatten() []*FileNode {
	t.FlatList = t.FlatList[:0] // Reuse slice
	if t.Root != nil {
		t.flattenNode(t.Root)
	}
	return t.FlatList
}

func (t *FileTree) flattenNode(node *FileNode) {
	for _, child := range node.Children {
		t.FlatList = append(t.FlatList, child)
		if child.IsDir && child.IsExpanded {
			t.flattenNode(child)
		}
	}
}

// GetNode returns the node at the given index, or nil if out of bounds.
func (t *FileTree) GetNode(index int) *FileNode {
	if index < 0 || index >= len(t.FlatList) {
		return nil
	}
	return t.FlatList[index]
}

// Len returns the number of visible nodes.
func (t *FileTree) Len() int {
	return len(t.FlatList)
}

// FindParentDir returns the parent directory node, or nil if at root.
func (t *FileTree) FindParentDir(node *FileNode) *FileNode {
	if node == nil || node.Parent == nil || node.Parent == t.Root {
		return nil
	}
	return node.Parent
}

// IndexOf returns the index of a node in the flat list, or -1 if not found.
func (t *FileTree) IndexOf(node *FileNode) int {
	for i, n := range t.FlatList {
		if n == node {
			return i
		}
	}
	return -1
}

// GetExpandedPaths returns the paths of all expanded directories.
func (t *FileTree) GetExpandedPaths() map[string]bool {
	expanded := make(map[string]bool)
	if t.Root != nil {
		t.collectExpanded(t.Root, expanded)
	}
	return expanded
}

func (t *FileTree) collectExpanded(node *FileNode, expanded map[string]bool) {
	for _, child := range node.Children {
		if child.IsDir && child.IsExpanded {
			expanded[child.Path] = true
			t.collectExpanded(child, expanded)
		}
	}
}

// RestoreExpandedPaths expands directories that were previously expanded.
func (t *FileTree) RestoreExpandedPaths(paths map[string]bool) {
	if t.Root == nil || len(paths) == 0 {
		return
	}
	t.restoreExpanded(t.Root, paths)
	t.Flatten()
}

func (t *FileTree) restoreExpanded(node *FileNode, paths map[string]bool) {
	for _, child := range node.Children {
		if child.IsDir && paths[child.Path] {
			// Load children if needed and expand
			if len(child.Children) == 0 {
				_ = t.loadChildren(child)
			}
			child.IsExpanded = true
			t.restoreExpanded(child, paths)
		}
	}
}

// Refresh reloads the tree from disk, preserving expanded state.
func (t *FileTree) Refresh() error {
	// Save expanded state before rebuild
	expandedPaths := t.GetExpandedPaths()

	// Rebuild tree
	if err := t.Build(); err != nil {
		return err
	}

	// Restore expanded state
	t.RestoreExpandedPaths(expandedPaths)
	return nil
}
