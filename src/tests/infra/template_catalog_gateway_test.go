package infra_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	infratemplate "github.com/joleques/northstar-ai/src/infra/template"
)

func TestTemplateCatalogGatewayLoad(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	toolsDir := filepath.Join(root, "tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	skillA := "type: skill\ncategories:\n  - software-architecture\nname: skill-a\ndescription: Skill A description\ninstructions: |\n  Execute Skill A.\n"
	if err := os.WriteFile(filepath.Join(toolsDir, "skill-a.yaml"), []byte(skillA), 0o644); err != nil {
		t.Fatal(err)
	}
	skillB := "type: skill\ncategories:\n  - documentation\nname: skill-b\ndescription: Skill B description\ninstructions: |\n  Execute Skill B.\n"
	if err := os.WriteFile(filepath.Join(toolsDir, "skill-b.yaml"), []byte(skillB), 0o644); err != nil {
		t.Fatal(err)
	}

	assistant := "type: assitent\ncategories:\n  - documentation\nid: test-assistant\nname: Test Assistant\ndescription: desc\ninstructions: This instructions text is long enough to pass validation.\nskills:\n  - skill-a\n"
	if err := os.WriteFile(filepath.Join(toolsDir, "test-assistant.yaml"), []byte(assistant), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("agents"), 0o644); err != nil {
		t.Fatal(err)
	}

	gateway := infratemplate.NewCatalogGateway(root)
	catalog, err := gateway.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(catalog.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(catalog.Skills))
	}
	if catalog.Skills[0].Contract == nil {
		t.Fatal("expected skill-a contract to be loaded")
	}
	if catalog.Skills[0].Contract.Name != "skill-a" {
		t.Fatalf("expected contract name skill-a, got %q", catalog.Skills[0].Contract.Name)
	}
	if catalog.Skills[1].Contract == nil {
		t.Fatal("expected skill-b contract to be loaded")
	}
	if len(catalog.Skills[0].Categories) == 0 || catalog.Skills[0].Categories[0] != "software-architecture" {
		t.Fatalf("expected skill categories to be loaded, got %#v", catalog.Skills[0].Categories)
	}

	if len(catalog.Assistants) != 1 {
		t.Fatalf("expected 1 assistant, got %d", len(catalog.Assistants))
	}

	if catalog.Assistants[0].ID != "test-assistant" {
		t.Fatalf("expected assistant id test-assistant, got %q", catalog.Assistants[0].ID)
	}

	if catalog.Assistants[0].Name != "Test Assistant" {
		t.Fatalf("expected assistant name Test Assistant, got %q", catalog.Assistants[0].Name)
	}

	if catalog.Assistants[0].Description != "desc" {
		t.Fatalf("expected assistant description desc, got %q", catalog.Assistants[0].Description)
	}

	if len(catalog.Assistants[0].Skills) != 1 || catalog.Assistants[0].Skills[0] != "skill-a" {
		t.Fatalf("expected assistant skills to be loaded, got %#v", catalog.Assistants[0].Skills)
	}
	if len(catalog.Assistants[0].Categories) != 1 || catalog.Assistants[0].Categories[0] != "documentation" {
		t.Fatalf("expected assistant categories to be loaded, got %#v", catalog.Assistants[0].Categories)
	}

	if catalog.AgentsTemplatePath == "" {
		t.Fatal("expected AGENTS template path to be set")
	}
}

func TestTemplateCatalogGatewayLoadPrefersClientTemplate(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(runtimeRoot, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	runtimeAssistant := "type: assitent\ncategories:\n  - platform\nid: runtime-assistant\nname: Runtime Assistant\ndescription: desc\ninstructions: Runtime instructions long enough for validation in tests.\nskills: []\n"
	if err := os.WriteFile(filepath.Join(runtimeRoot, "tools", "runtime-assistant.yaml"), []byte(runtimeAssistant), 0o644); err != nil {
		t.Fatal(err)
	}

	clientRoot := t.TempDir()
	clientToolsDir := filepath.Join(clientRoot, ".northstar", "template", "tools")
	if err := os.MkdirAll(clientToolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	clientAssistant := "type: assitent\ncategories:\n  - platform\nid: client-assistant\nname: Client Assistant\ndescription: desc\ninstructions: Client instructions long enough for validation in tests.\nskills: []\n"
	if err := os.WriteFile(filepath.Join(clientToolsDir, "client-assistant.yaml"), []byte(clientAssistant), 0o644); err != nil {
		t.Fatal(err)
	}

	gateway := infratemplate.NewCatalogGateway(runtimeRoot)
	catalog, err := gateway.Load(context.Background(), clientRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(catalog.Assistants) != 1 || catalog.Assistants[0].ID != "client-assistant" {
		t.Fatalf("expected catalog from client template, got %#v", catalog.Assistants)
	}
}

func TestTemplateCatalogGatewayLoadSetsSkillSourceDirFromSiblingAssetsDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	toolsDir := filepath.Join(root, "tools")
	if err := os.MkdirAll(filepath.Join(toolsDir, "nina-designer", "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}

	skillA := "type: skill\ncategories:\n  - media\nname: nina-designer\ndescription: Skill A description\ninstructions: |\n  Execute Skill A.\n"
	if err := os.WriteFile(filepath.Join(toolsDir, "nina-designer.yaml"), []byte(skillA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolsDir, "nina-designer", "scripts", "run.sh"), []byte("#!/bin/sh\necho ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	gateway := infratemplate.NewCatalogGateway(root)
	catalog, err := gateway.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(catalog.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(catalog.Skills))
	}

	expectedDir := filepath.Join(toolsDir, "nina-designer")
	if catalog.Skills[0].SourceDir != expectedDir {
		t.Fatalf("expected SourceDir %q, got %q", expectedDir, catalog.Skills[0].SourceDir)
	}
}

func TestTemplateCatalogGatewayLoadFallsBackToRuntimeWhenClientTemplateMissing(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	runtimeToolsDir := filepath.Join(runtimeRoot, "tools")
	if err := os.MkdirAll(runtimeToolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runtimeAssistant := "type: assitent\ncategories:\n  - platform\nid: runtime-assistant\nname: Runtime Assistant\ndescription: desc\ninstructions: Runtime instructions long enough for validation in tests.\nskills: []\n"
	if err := os.WriteFile(filepath.Join(runtimeToolsDir, "runtime-assistant.yaml"), []byte(runtimeAssistant), 0o644); err != nil {
		t.Fatal(err)
	}

	clientRoot := t.TempDir()
	gateway := infratemplate.NewCatalogGateway(runtimeRoot)
	catalog, err := gateway.Load(context.Background(), clientRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(catalog.Assistants) != 1 || catalog.Assistants[0].ID != "runtime-assistant" {
		t.Fatalf("expected fallback catalog from runtime template, got %#v", catalog.Assistants)
	}
}

func TestTemplateCatalogGatewayLoadDefaultTemplateIncludesGovernanceTools(t *testing.T) {
	t.Parallel()

	gateway := infratemplate.NewCatalogGateway(filepath.Join("..", "..", "templates", "default"))
	catalog, err := gateway.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	names := make([]string, 0, len(catalog.Skills))
	for _, skill := range catalog.Skills {
		names = append(names, skill.Name)
	}

	for _, expected := range []string{
		"triagem-demanda",
		"plano-implementacao",
		"quality-assurance",
		"arquitetura-revisor",
		"software-principles-revisor",
	} {
		if !slices.Contains(names, expected) {
			t.Fatalf("expected default catalog to include %q, got %#v", expected, names)
		}
	}
}
