package infra_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	infrainstall "github.com/joleques/northstar-ai/src/infra/install"
	usecase "github.com/joleques/northstar-ai/src/use_case"
)

func TestFilesystemGatewaySaveProjectContext(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	docSource := filepath.Join(outputDir, "README.md")
	if err := os.WriteFile(docSource, []byte("# Docs"), 0o644); err != nil {
		t.Fatal(err)
	}

	gateway := infrainstall.NewFilesystemGateway()
	result, err := gateway.SaveProjectContext(context.Background(), usecase.StartRequest{
		Target:        "codex",
		Title:         "Heimdall",
		Description:   "Contexto base do projeto.",
		Documentation: []string{docSource, "Arquitetura em evolucao"},
		OutputDir:     outputDir,
	})
	if err != nil {
		t.Fatalf("expected save project context to succeed, got %v", err)
	}

	if len(result.Created) < 3 {
		t.Fatalf("expected at least 3 created entries, got %d", len(result.Created))
	}

	manifestPath := filepath.Join(outputDir, ".heimdall", "context", "project-context.yaml")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("expected manifest to exist, got %v", err)
	}

	for _, fragment := range []string{
		"target: codex",
		"project_root: " + outputDir,
		"title: Heimdall",
		"stored_path: docs/01-README.md",
		"kind: note",
	} {
		if !strings.Contains(string(content), fragment) {
			t.Fatalf("expected manifest to contain %q, got %s", fragment, string(content))
		}
	}

	if _, err := os.Stat(filepath.Join(outputDir, ".heimdall", "context", "docs", "01-README.md")); err != nil {
		t.Fatalf("expected copied documentation to exist, got %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputDir, ".heimdall", "context", "docs", "02-note.md")); err != nil {
		t.Fatalf("expected note documentation to exist, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, ".heimdall", "context", "README.md")); !os.IsNotExist(err) {
		t.Fatalf("expected context README to not be generated, got err=%v", err)
	}
}

func TestFilesystemGatewaySaveProjectContextIdempotent(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway()
	request := usecase.StartRequest{
		Target:        "codex",
		Title:         "Heimdall",
		Description:   "Contexto base do projeto.",
		Documentation: []string{"README.md"},
		OutputDir:     outputDir,
	}

	first, err := gateway.SaveProjectContext(context.Background(), request)
	if err != nil {
		t.Fatalf("expected first save to succeed, got %v", err)
	}

	second, err := gateway.SaveProjectContext(context.Background(), request)
	if err != nil {
		t.Fatalf("expected second save to succeed, got %v", err)
	}

	if len(first.Created) == 0 {
		t.Fatal("expected created entries on first save")
	}

	if len(second.Skipped) == 0 {
		t.Fatal("expected skipped entries on second save")
	}
	if len(second.Updated) == 0 {
		t.Fatal("expected updated entries on second save")
	}
}

func TestFilesystemGatewaySaveProjectContextUpdatesManifestValues(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway()

	_, err := gateway.SaveProjectContext(context.Background(), usecase.StartRequest{
		Target:      "codex",
		Title:       "Primeiro Titulo",
		Description: "Primeira descricao",
		OutputDir:   outputDir,
	})
	if err != nil {
		t.Fatalf("expected first save to succeed, got %v", err)
	}

	_, err = gateway.SaveProjectContext(context.Background(), usecase.StartRequest{
		Target:      "codex",
		Title:       "Segundo Titulo",
		Description: "Descricao atualizada",
		OutputDir:   outputDir,
	})
	if err != nil {
		t.Fatalf("expected second save to succeed, got %v", err)
	}

	manifestPath := filepath.Join(outputDir, ".heimdall", "context", "project-context.yaml")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("expected manifest to exist, got %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "title: Segundo Titulo") {
		t.Fatalf("expected updated title in manifest, got %s", text)
	}
	if !strings.Contains(text, "description: Descricao atualizada") {
		t.Fatalf("expected updated description in manifest, got %s", text)
	}
}

func TestFilesystemGatewayLoadProjectContext(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway()

	_, err := gateway.SaveProjectContext(context.Background(), usecase.StartRequest{
		Target:        "cursor",
		Title:         "Heimdall",
		Description:   "Contexto base do projeto.",
		Documentation: []string{"README.md"},
		OutputDir:     outputDir,
	})
	if err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	projectContext, err := gateway.LoadProjectContext(context.Background(), outputDir)
	if err != nil {
		t.Fatalf("expected load project context to succeed, got %v", err)
	}

	if projectContext.Target != "cursor" {
		t.Fatalf("expected target cursor, got %q", projectContext.Target)
	}
	if projectContext.ProjectRoot != outputDir {
		t.Fatalf("expected project root %q, got %q", outputDir, projectContext.ProjectRoot)
	}
}

func TestFilesystemGatewayLoadProjectContextWithOnlyTarget(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outputDir, ".heimdall", "context"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, ".heimdall", "context", "project-context.yaml"), []byte("target: codex\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	gateway := infrainstall.NewFilesystemGateway()
	projectContext, err := gateway.LoadProjectContext(context.Background(), outputDir)
	if err != nil {
		t.Fatalf("expected load project context with only target to succeed, got %v", err)
	}

	if projectContext.Target != "codex" {
		t.Fatalf("expected target codex, got %q", projectContext.Target)
	}
	if projectContext.ProjectRoot != outputDir {
		t.Fatalf("expected project root fallback %q, got %q", outputDir, projectContext.ProjectRoot)
	}
}
