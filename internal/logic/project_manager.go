package logic

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ProjectItem struct {
	Path     string
	Name     string
	IsDir    bool
	Children []*ProjectItem
}

type ProjectManager struct {
	RootPath     string
	IsActive     bool
	ContextFiles map[string]bool
}

func NewProjectManager() *ProjectManager {
	return &ProjectManager{
		ContextFiles: make(map[string]bool),
	}
}

func (pm *ProjectManager) SetRootPath(path string) {
	pm.RootPath = path
	pm.IsActive = true
}

func (pm *ProjectManager) GetProjectTree() (*ProjectItem, error) {
	return pm.buildTree(pm.RootPath)
}

func (pm *ProjectManager) buildTree(path string) (*ProjectItem, error) {
	info, err := os.Stat(path)
	if err != nil { return nil, err }

	item := &ProjectItem{
		Path:  path,
		Name:  info.Name(),
		IsDir: info.IsDir(),
	}

	if !info.IsDir() { return item, nil }

	entries, err := os.ReadDir(path)
	if err != nil { return item, nil }

	// Separate and sort
	var dirs, files []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "vendor" { continue }

		if entry.IsDir() {
			dirs = append(dirs, entry)
		} else {
			// Show all files in IDE mode, or filter if desired
			files = append(files, entry)
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	for _, d := range dirs {
		child, _ := pm.buildTree(filepath.Join(path, d.Name()))
		if child != nil { item.Children = append(item.Children, child) }
	}
	for _, f := range files {
		item.Children = append(item.Children, &ProjectItem{
			Path:  filepath.Join(path, f.Name()),
			Name:  f.Name(),
			IsDir: false,
		})
	}

	return item, nil
}

func (pm *ProjectManager) IsFileInProject(path string) bool {
	if !pm.IsActive { return false }
	rel, err := filepath.Rel(pm.RootPath, path)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// Context Logic (Simple Set)
func (pm *ProjectManager) ToggleContextFile(path string) bool {
	if pm.ContextFiles[path] {
		delete(pm.ContextFiles, path)
		return false
	}
	pm.ContextFiles[path] = true
	return true
}

func (pm *ProjectManager) IsFileInContext(path string) bool {
	return pm.ContextFiles[path]
}

func (pm *ProjectManager) GetContextFiles() []string {
	res := []string{}
	for k := range pm.ContextFiles { res = append(res, k) }
	return res
}

// ClearContextFiles очищает список файлов, используемых в качестве контекста для AI.
func (pm *ProjectManager) ClearContextFiles() {
	// Поле называется ContextFiles и является картой (map).
	// Для очистки мы пересоздаем ее.
	pm.ContextFiles = make(map[string]bool)
}
