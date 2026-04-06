package infra_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heimdall-app/heimdall/src/domain"
	infrainstall "github.com/heimdall-app/heimdall/src/infra/install"
	usecase "github.com/heimdall-app/heimdall/src/use_case"
)

func TestFilesystemGatewayInstallSkillsAndAgentsPolicy(t *testing.T) {
	t.Parallel()

	sourceRoot := t.TempDir()
	skillSource := filepath.Join(sourceRoot, "skill-a")
	if err := os.MkdirAll(skillSource, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSource, "SKILL.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	agentsTemplate := filepath.Join(sourceRoot, "AGENTS.md")
	if err := os.WriteFile(agentsTemplate, []byte("agents"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway()
	request := usecase.InstallRequest{
		Target:       domain.TargetCodex,
		OutputDir:    outputDir,
		AgentsPolicy: domain.AgentsPolicyIfMissing,
	}

	first, err := gateway.InstallSkills(context.Background(), request, []usecase.SkillAsset{{Name: "skill-a", SourceDir: skillSource}})
	if err != nil {
		t.Fatalf("expected first install to succeed, got %v", err)
	}
	if len(first.Installed) != 1 {
		t.Fatalf("expected 1 installed skill, got %d", len(first.Installed))
	}

	second, err := gateway.InstallSkills(context.Background(), request, []usecase.SkillAsset{{Name: "skill-a", SourceDir: skillSource}})
	if err != nil {
		t.Fatalf("expected second install to succeed, got %v", err)
	}
	if len(second.Skipped) != 1 {
		t.Fatalf("expected 1 skipped skill on second install, got %d", len(second.Skipped))
	}

	agentsFirst, err := gateway.ApplyAgentsPolicy(context.Background(), request, agentsTemplate)
	if err != nil {
		t.Fatalf("expected first agents policy to succeed, got %v", err)
	}
	if len(agentsFirst.Installed) != 1 {
		t.Fatalf("expected agents to be installed once, got %d", len(agentsFirst.Installed))
	}

	agentsSecond, err := gateway.ApplyAgentsPolicy(context.Background(), request, agentsTemplate)
	if err != nil {
		t.Fatalf("expected second agents policy to succeed, got %v", err)
	}
	if len(agentsSecond.Skipped) != 1 {
		t.Fatalf("expected agents to be skipped when already exists, got %d", len(agentsSecond.Skipped))
	}
}

func TestFilesystemGatewayApplyAgentsPolicyRendersAgentsTemplateWithProjectContext(t *testing.T) {
	t.Parallel()

	sourceRoot := t.TempDir()
	agentsTemplate := filepath.Join(sourceRoot, "AGENTS.md")
	templateContent := strings.Join([]string{
		"title={{PROJECT_TITLE}}",
		"description={{PROJECT_DESCRIPTION}}",
		"target={{TARGET_PLATFORM}}",
		"root={{PROJECT_ROOT}}",
		"docs:",
		"{{PROJECT_DOCS_SUMMARY}}",
		"persona={{SQUAD_PERSONA}}",
	}, "\n")
	if err := os.WriteFile(agentsTemplate, []byte(templateContent), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway()
	_, err := gateway.SaveProjectContext(context.Background(), usecase.StartRequest{
		Target:        domain.TargetCodex,
		Title:         "Heimdall Commerce",
		Description:   "Squad para operacao comercial e melhoria de funil.",
		Documentation: []string{"playbook-vendas.md", "metas-q2.md", "dados-pipeline.csv", "faq-comercial.md"},
		OutputDir:     outputDir,
	})
	if err != nil {
		t.Fatalf("expected save project context to succeed, got %v", err)
	}

	result, err := gateway.ApplyAgentsPolicy(context.Background(), usecase.InstallRequest{
		Target:       domain.TargetCodex,
		OutputDir:    outputDir,
		AgentsPolicy: domain.AgentsPolicyOverwrite,
	}, agentsTemplate)
	if err != nil {
		t.Fatalf("expected agents policy to succeed, got %v", err)
	}
	if len(result.Installed) != 1 {
		t.Fatalf("expected agents to be installed, got %#v", result)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("expected generated AGENTS.md to exist, got %v", err)
	}

	text := string(content)
	for _, fragment := range []string{
		"title=Heimdall Commerce",
		"description=Squad para operacao comercial e melhoria de funil.",
		"target=codex",
		"root=" + outputDir,
		"- playbook-vendas.md",
		"- metas-q2.md",
		"- dados-pipeline.csv",
		"- ... e mais 1 fonte(s) de contexto.",
		"persona=Lideranca de operacao comercial orientada a processo, clareza de funil e melhoria continua por dados.",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected rendered AGENTS.md to contain %q, got %s", fragment, text)
		}
	}

	if strings.Contains(text, "{{PROJECT_TITLE}}") {
		t.Fatalf("expected placeholders to be rendered, got %s", text)
	}
}

func TestFilesystemGatewayApplyAgentsPolicyRendersFallbackWhenContextIsMissing(t *testing.T) {
	t.Parallel()

	sourceRoot := t.TempDir()
	agentsTemplate := filepath.Join(sourceRoot, "AGENTS.md")
	templateContent := strings.Join([]string{
		"title={{PROJECT_TITLE}}",
		"description={{PROJECT_DESCRIPTION}}",
		"target={{TARGET_PLATFORM}}",
		"root={{PROJECT_ROOT}}",
		"docs:",
		"{{PROJECT_DOCS_SUMMARY}}",
		"persona={{SQUAD_PERSONA}}",
	}, "\n")
	if err := os.WriteFile(agentsTemplate, []byte(templateContent), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway()

	_, err := gateway.ApplyAgentsPolicy(context.Background(), usecase.InstallRequest{
		Target:       domain.TargetCodex,
		OutputDir:    outputDir,
		AgentsPolicy: domain.AgentsPolicyOverwrite,
	}, agentsTemplate)
	if err != nil {
		t.Fatalf("expected agents policy to succeed without project context, got %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("expected generated AGENTS.md to exist, got %v", err)
	}

	text := string(content)
	for _, fragment := range []string{
		"title=Projeto sem titulo informado",
		"description=Contexto de negocio nao informado no heimdall start.",
		"target=nao-definido",
		"root=" + outputDir,
		"- Nenhuma documentacao registrada no heimdall start.",
		"persona=Lideranca de squad multidisciplinar orientada a problema, com fronteiras claras de responsabilidade e colaboracao entre especialistas.",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected fallback AGENTS.md to contain %q, got %s", fragment, text)
		}
	}
}

func TestFilesystemGatewayInitTarget(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway()

	result, err := gateway.InitTarget(context.Background(), usecase.InitRequest{
		Target:       domain.TargetClaude,
		OutputDir:    outputDir,
		AgentsPolicy: domain.AgentsPolicySkip,
	})
	if err != nil {
		t.Fatalf("expected init to succeed, got %v", err)
	}

	if len(result.Created) < 2 {
		t.Fatalf("expected at least 2 created directories, got %d", len(result.Created))
	}

	if _, err := os.Stat(filepath.Join(outputDir, ".claude", "skills")); err != nil {
		t.Fatalf("expected .claude/skills to exist, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, ".claude", "assistants")); err == nil {
		t.Fatal("expected .claude/assistants to not be created")
	}

	manifestPath := filepath.Join(outputDir, ".heimdall", "context", "project-context.yaml")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("expected project-context.yaml to exist after init, got %v", err)
	}
	if !strings.Contains(string(content), "target: claude") {
		t.Fatalf("expected init to persist target in project-context.yaml, got %s", string(content))
	}
	if !strings.Contains(string(content), "project_root: "+outputDir) {
		t.Fatalf("expected init to persist project_root in project-context.yaml, got %s", string(content))
	}

	gitignorePath := filepath.Join(outputDir, ".gitignore")
	gitignoreContent, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("expected .gitignore to exist after init, got %v", err)
	}
	if !strings.Contains(string(gitignoreContent), ".heimdall") {
		t.Fatalf("expected .gitignore to contain .heimdall, got %s", string(gitignoreContent))
	}
}

func TestFilesystemGatewayInitTargetAppendsHeimdallToExistingGitignoreWithoutDuplicates(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outputDir, ".gitignore"), []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	gateway := infrainstall.NewFilesystemGateway()
	_, err := gateway.InitTarget(context.Background(), usecase.InitRequest{
		Target:       domain.TargetCodex,
		OutputDir:    outputDir,
		AgentsPolicy: domain.AgentsPolicySkip,
	})
	if err != nil {
		t.Fatalf("expected first init to succeed, got %v", err)
	}

	_, err = gateway.InitTarget(context.Background(), usecase.InitRequest{
		Target:       domain.TargetCodex,
		OutputDir:    outputDir,
		AgentsPolicy: domain.AgentsPolicySkip,
	})
	if err != nil {
		t.Fatalf("expected second init to succeed, got %v", err)
	}

	gitignoreContent, err := os.ReadFile(filepath.Join(outputDir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to exist, got %v", err)
	}

	text := string(gitignoreContent)
	if !strings.Contains(text, "node_modules/") {
		t.Fatalf("expected existing .gitignore content to be preserved, got %s", text)
	}
	if strings.Count(text, ".heimdall") != 1 {
		t.Fatalf("expected .heimdall to be present exactly once, got %s", text)
	}
}

func TestFilesystemGatewayInitTargetCopiesTemplateSnapshotToHeimdall(t *testing.T) {
	t.Parallel()

	templateRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(templateRoot, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateRoot, "AGENTS.md"), []byte("agents-template"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateRoot, "tools", "tool-a.yaml"), []byte("type: skill\nname: tool-a\ndescription: desc\ninstructions: run"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway(filepath.Join(templateRoot, "AGENTS.md"))

	result, err := gateway.InitTarget(context.Background(), usecase.InitRequest{
		Target:       domain.TargetCodex,
		OutputDir:    outputDir,
		AgentsPolicy: domain.AgentsPolicySkip,
	})
	if err != nil {
		t.Fatalf("expected init to succeed, got %v", err)
	}

	if len(result.Created) < 3 {
		t.Fatalf("expected at least 3 created entries (.codex dirs + template snapshot), got %d", len(result.Created))
	}

	if _, err := os.Stat(filepath.Join(outputDir, ".heimdall", "template", "AGENTS.md")); err != nil {
		t.Fatalf("expected .heimdall/template/AGENTS.md to exist, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, ".heimdall", "template", "tools", "tool-a.yaml")); err != nil {
		t.Fatalf("expected .heimdall/template/tools/tool-a.yaml to exist, got %v", err)
	}
}

func TestFilesystemGatewayInitTargetAutoInstallsPlatformTools(t *testing.T) {
	t.Parallel()

	templateRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(templateRoot, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateRoot, "AGENTS.md"), []byte("agents-template"), 0o644); err != nil {
		t.Fatal(err)
	}

	platformSkill := `type: skill
categories:
  - platform
name: platform-helper
description: Platform helper skill.
instructions: |
  # Platform Helper
  Execute platform helper routines.`
	if err := os.WriteFile(filepath.Join(templateRoot, "tools", "platform-helper.yaml"), []byte(platformSkill), 0o644); err != nil {
		t.Fatal(err)
	}

	platformAssistant := `type: assitent
categories:
  - platform
id: heimdall-list-lib
name: List Library
description: Executes list-lib command for platform discovery.
instructions: |
  Execute heimdall list-lib and return exact results for the user.
skills: []
tools:
  - shell`
	if err := os.WriteFile(filepath.Join(templateRoot, "tools", "heimdall-list-lib.yaml"), []byte(platformAssistant), 0o644); err != nil {
		t.Fatal(err)
	}

	nonPlatformAssistant := `type: assitent
categories:
  - documentation
id: doc-api
name: Doc API
description: Documentation workflow.
instructions: |
  Generate API documentation with review workflow.
skills: []
tools:
  - shell`
	if err := os.WriteFile(filepath.Join(templateRoot, "tools", "doc-api.yaml"), []byte(nonPlatformAssistant), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway(filepath.Join(templateRoot, "AGENTS.md"))

	_, err := gateway.InitTarget(context.Background(), usecase.InitRequest{
		Target:       domain.TargetCodex,
		OutputDir:    outputDir,
		AgentsPolicy: domain.AgentsPolicySkip,
	})
	if err != nil {
		t.Fatalf("expected init to succeed, got %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputDir, ".codex", "skills", "platform-helper", "SKILL.md")); err != nil {
		t.Fatalf("expected platform skill platform-helper to be auto-installed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, ".codex", "skills", "heimdall-list-lib", "SKILL.md")); err != nil {
		t.Fatalf("expected platform assistant heimdall-list-lib to be materialized as skill markdown, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, ".codex", "assistants", "heimdall-list-lib.yaml")); err == nil {
		t.Fatal("expected no assistant yaml for auto-installed platform assistant")
	}
	if _, err := os.Stat(filepath.Join(outputDir, ".codex", "assistants", "doc-api.yaml")); err == nil {
		t.Fatal("expected non-platform assistant doc-api.yaml to not be auto-installed")
	}
}

func TestFilesystemGatewayInstallAssistantsCreatesCodexWrapper(t *testing.T) {
	t.Parallel()

	sourceRoot := t.TempDir()
	assistantSource := filepath.Join(sourceRoot, "write-tech-article.yaml")
	if err := os.WriteFile(assistantSource, []byte("id: write-tech-article"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway()

	result, err := gateway.InstallAssistants(context.Background(), usecase.InstallRequest{
		Target:    domain.TargetCodex,
		OutputDir: outputDir,
	}, []usecase.AssistantAsset{
		{
			ID:           "write-tech-article",
			Name:         "Write Tech Article",
			Description:  "Orquestra escrita e revisao de artigo tecnico.",
			Instructions: "Conduza pesquisa, escrita e revisao em ciclo iterativo.",
			SourcePath:   assistantSource,
			Skills:       []string{"engineering-writer", "engineering-writer-revisor"},
		},
	})
	if err != nil {
		t.Fatalf("expected install assistants to succeed, got %v", err)
	}

	if len(result.Installed) != 1 || result.Installed[0] != "assistant:write-tech-article" {
		t.Fatalf("expected assistant wrapper install result, got %#v", result.Installed)
	}

	wrapperPath := filepath.Join(outputDir, ".codex", "skills", "assistant-write-tech-article", "SKILL.md")
	content, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("expected wrapper skill file to exist, got %v", err)
	}

	if !strings.Contains(string(content), "Conduza pesquisa, escrita e revisao em ciclo iterativo.") {
		t.Fatalf("expected wrapper instructions content, got %s", string(content))
	}
}

func TestFilesystemGatewayInstallSkillsMaterializesSkillMarkdownFromContract(t *testing.T) {
	t.Parallel()

	sourceRoot := t.TempDir()
	skillSource := filepath.Join(sourceRoot, "skill-a")
	if err := os.MkdirAll(skillSource, 0o755); err != nil {
		t.Fatal(err)
	}

	contract := "name: Skill A\ndescription: Skill A description\ninstructions: |\n  # Skill A\n\n  Execute Skill A.\n"
	if err := os.WriteFile(filepath.Join(skillSource, "skill.yaml"), []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway()
	request := usecase.InstallRequest{
		Target:    domain.TargetCodex,
		OutputDir: outputDir,
	}

	result, err := gateway.InstallSkills(context.Background(), request, []usecase.SkillAsset{
		{
			Name:      "skill-a",
			SourceDir: skillSource,
			Contract: &usecase.SkillContract{
				Name:         "Skill A",
				Description:  "Skill A description",
				Instructions: "# Skill A\n\nExecute Skill A.",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected install skills to succeed, got %v", err)
	}
	if len(result.Installed) != 1 {
		t.Fatalf("expected 1 installed skill, got %#v", result)
	}

	skillPath := filepath.Join(outputDir, ".codex", "skills", "skill-a", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("expected generated SKILL.md file to exist, got %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "name: Skill A") {
		t.Fatalf("expected generated SKILL.md header, got %s", text)
	}
	if !strings.Contains(text, "Execute Skill A.") {
		t.Fatalf("expected generated SKILL.md instructions, got %s", text)
	}
}

func TestFilesystemGatewayUpdateAppRefreshesPlatformSkills(t *testing.T) {
	t.Parallel()

	templateRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(templateRoot, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateRoot, "AGENTS.md"), []byte("agents-template"), 0o644); err != nil {
		t.Fatal(err)
	}

	currentPlatformTool := `type: skill
categories:
  - platform
name: heimdall-install
description: Install helper.
instructions: |
  execute install helper`
	if err := os.WriteFile(filepath.Join(templateRoot, "tools", "heimdall-install.yaml"), []byte(currentPlatformTool), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outputDir, ".codex", "skills", "heimdall-old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, ".codex", "skills", "heimdall-old", "SKILL.md"), []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(outputDir, ".heimdall", "template", "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	previousPlatformTool := `type: skill
categories:
  - platform
name: heimdall-old
description: Old helper.
instructions: |
  legacy helper`
	if err := os.WriteFile(filepath.Join(outputDir, ".heimdall", "template", "tools", "heimdall-old.yaml"), []byte(previousPlatformTool), 0o644); err != nil {
		t.Fatal(err)
	}
	nonPlatformTool := `type: skill
categories:
  - documentation
name: user-custom-doc
description: User custom tool.
instructions: |
  custom docs flow`
	if err := os.WriteFile(filepath.Join(outputDir, ".heimdall", "template", "tools", "user-custom-doc.yaml"), []byte(nonPlatformTool), 0o644); err != nil {
		t.Fatal(err)
	}

	gateway := infrainstall.NewFilesystemGateway(filepath.Join(templateRoot, "AGENTS.md"))
	result, err := gateway.UpdateApp(context.Background(), usecase.UpdateAppRequest{
		Target:    domain.TargetCodex,
		OutputDir: outputDir,
	})
	if err != nil {
		t.Fatalf("expected update-app to succeed, got %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputDir, ".codex", "skills", "heimdall-old")); err == nil {
		t.Fatal("expected legacy platform skill to be removed")
	}
	if _, err := os.Stat(filepath.Join(outputDir, ".codex", "skills", "heimdall-install", "SKILL.md")); err != nil {
		t.Fatalf("expected current platform skill to be installed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, ".heimdall", "template", "tools", "user-custom-doc.yaml")); err != nil {
		t.Fatalf("expected non-platform custom tool to be preserved, got %v", err)
	}

	if len(result.Removed) == 0 {
		t.Fatalf("expected removed entries, got %#v", result)
	}
	if len(result.Installed) == 0 {
		t.Fatalf("expected installed entries, got %#v", result)
	}
}

func TestFilesystemGatewayUpdateAppRefreshesManagedNonPlatformSkill(t *testing.T) {
	t.Parallel()

	templateRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(templateRoot, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateRoot, "AGENTS.md"), []byte("agents-template"), 0o644); err != nil {
		t.Fatal(err)
	}

	toolV1 := `type: skill
categories:
  - documentation
name: engineering-writer
description: Writer helper.
instructions: |
  v1 instructions`
	if err := os.WriteFile(filepath.Join(templateRoot, "tools", "engineering-writer.yaml"), []byte(toolV1), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	gateway := infrainstall.NewFilesystemGateway(filepath.Join(templateRoot, "AGENTS.md"))
	installResult, err := gateway.InstallSkills(context.Background(), usecase.InstallRequest{
		Target:    domain.TargetCodex,
		OutputDir: outputDir,
		Force:     true,
	}, []usecase.SkillAsset{
		{
			Name: "engineering-writer",
			Contract: &usecase.SkillContract{
				Name:         "engineering-writer",
				Description:  "Writer helper.",
				Instructions: "v1 instructions",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected skill install to succeed, got %v", err)
	}
	if len(installResult.Installed) == 0 {
		t.Fatalf("expected installed skill, got %#v", installResult)
	}

	toolV2 := `type: skill
categories:
  - documentation
name: engineering-writer
description: Writer helper.
instructions: |
  v2 instructions`
	if err := os.WriteFile(filepath.Join(templateRoot, "tools", "engineering-writer.yaml"), []byte(toolV2), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = gateway.UpdateApp(context.Background(), usecase.UpdateAppRequest{
		Target:    domain.TargetCodex,
		OutputDir: outputDir,
	})
	if err != nil {
		t.Fatalf("expected update-app to succeed, got %v", err)
	}

	skillPath := filepath.Join(outputDir, ".codex", "skills", "engineering-writer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("expected managed non-platform skill to exist, got %v", err)
	}
	if !strings.Contains(string(content), "v2 instructions") {
		t.Fatalf("expected managed non-platform skill to be updated to v2, got %s", string(content))
	}
}
