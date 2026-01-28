package logic

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileManager handles file system operations without side effects on UI state.
type FileManager struct{}

func NewFileManager() *FileManager {
	return &FileManager{}
}

// ReadFile simply reads and returns content.
func (fm *FileManager) ReadFile(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}

// WriteFile writes content to path.
func (fm *FileManager) WriteFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	err := ioutil.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// RunGoFmt runs 'go fmt' on the file.
func (fm *FileManager) RunGoFmt(path string) error {
	cmd := exec.Command("go", "fmt", path)
	return cmd.Run()
}

// RenameFile renames a file or directory.
func (fm *FileManager) RenameFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// DeletePath removes a file or directory.
func (fm *FileManager) DeletePath(path string) error {
	return os.RemoveAll(path) 
}

// CollectSpecificFilesContext reads multiple files and formats them for the LLM.
func (fm *FileManager) CollectSpecificFilesContext(paths []string) (string, error) {
	var sb strings.Builder
	sb.WriteString("Project Context:\n")
	
	for _, path := range paths {
		content, err := fm.ReadFile(path)
		if err != nil {
			continue // Skip unreadable files
		}
		sb.WriteString(fmt.Sprintf("--- File: %s ---\n", filepath.Base(path)))
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}
	
	return sb.String(), nil
}
