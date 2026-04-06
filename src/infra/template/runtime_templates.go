package template

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	projecttemplates "github.com/joleques/northstar-ai/src/templates"
)

func ResolveTemplateRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	if root := firstExistingTemplateRoot(
		filepath.Join(cwd, "src", "templates", "default"),
		filepath.Join(cwd, "templates", "default"),
	); root != "" {
		return root, nil
	}

	executablePath, err := os.Executable()
	if err == nil {
		executableDir := filepath.Dir(executablePath)
		if root := firstExistingTemplateRoot(
			filepath.Join(executableDir, "src", "templates", "default"),
			filepath.Join(executableDir, "templates", "default"),
		); root != "" {
			return root, nil
		}
	}

	return extractEmbeddedTemplates()
}

func firstExistingTemplateRoot(candidates ...string) string {
	for _, candidate := range candidates {
		if isDir(candidate) {
			return candidate
		}
	}
	return ""
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func extractEmbeddedTemplates() (string, error) {
	cacheRoot := filepath.Join(os.TempDir(), "heimdall-runtime-templates")
	templateRoot := filepath.Join(cacheRoot, "default")

	if err := os.RemoveAll(cacheRoot); err != nil {
		return "", fmt.Errorf("reset runtime template cache: %w", err)
	}

	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return "", fmt.Errorf("create runtime template cache: %w", err)
	}

	const embeddedPrefix = "default"
	walkErr := fs.WalkDir(projecttemplates.DefaultTemplates, embeddedPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relativePath := strings.TrimPrefix(path, embeddedPrefix)
		relativePath = strings.TrimPrefix(relativePath, "/")
		destination := filepath.Join(templateRoot, relativePath)

		if d.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		content, readErr := fs.ReadFile(projecttemplates.DefaultTemplates, path)
		if readErr != nil {
			return readErr
		}

		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}

		return os.WriteFile(destination, content, 0o644)
	})
	if walkErr != nil {
		return "", fmt.Errorf("extract embedded templates: %w", walkErr)
	}

	return templateRoot, nil
}
