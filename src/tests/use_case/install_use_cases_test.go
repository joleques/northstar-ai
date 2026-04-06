package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/joleques/northstar-ai/src/domain"
	usecase "github.com/joleques/northstar-ai/src/use_case"
)

type fakeCatalogGateway struct {
	catalog usecase.Catalog
	err     error
}

func (f fakeCatalogGateway) Load(context.Context, string) (usecase.Catalog, error) {
	return f.catalog, f.err
}

type fakeInstallGateway struct {
	skillsInput     []usecase.SkillAsset
	assistantsInput []usecase.AssistantAsset
	agentsTemplate  string
	projectContext  domain.ProjectContext
	projectErr      error
	lastRequest     usecase.InstallRequest
}

func (f *fakeInstallGateway) InstallSkills(_ context.Context, request usecase.InstallRequest, skills []usecase.SkillAsset) (usecase.InstallResult, error) {
	f.lastRequest = request
	f.skillsInput = skills
	installed := make([]string, 0, len(skills))
	for _, skill := range skills {
		installed = append(installed, "skill:"+skill.Name)
	}
	return usecase.InstallResult{Installed: installed}, nil
}

func (f *fakeInstallGateway) InstallAssistants(_ context.Context, request usecase.InstallRequest, assistants []usecase.AssistantAsset) (usecase.InstallResult, error) {
	f.lastRequest = request
	f.assistantsInput = assistants
	installed := make([]string, 0, len(assistants))
	for _, assistant := range assistants {
		installed = append(installed, "assistant:"+assistant.ID)
	}
	return usecase.InstallResult{Installed: installed}, nil
}

func (f *fakeInstallGateway) ApplyAgentsPolicy(_ context.Context, _ usecase.InstallRequest, templateAgentsPath string) (usecase.InstallResult, error) {
	f.agentsTemplate = templateAgentsPath
	return usecase.InstallResult{Installed: []string{"agents"}}, nil
}

func (f *fakeInstallGateway) LoadProjectContext(_ context.Context, _ string) (domain.ProjectContext, error) {
	return f.projectContext, f.projectErr
}

func TestInstallAssistantUseCaseInstallsAssistantAndAssociatedSkills(t *testing.T) {
	t.Parallel()

	catalog := usecase.Catalog{
		Skills: []usecase.SkillAsset{
			{Name: "designer", SourceDir: "/tmp/designer"},
			{Name: "researcher", SourceDir: "/tmp/researcher"},
		},
		Assistants: []usecase.AssistantAsset{
			{ID: "instagram-post-studio", SourcePath: "/tmp/instagram-post-studio.yaml", Skills: []string{"researcher", "designer"}},
		},
		AgentsTemplatePath: "/tmp/AGENTS.md",
	}

	installGateway := &fakeInstallGateway{}
	installGateway.projectContext = domain.ProjectContext{Target: domain.TargetCodex, Title: "Heimdall", Description: "desc", Documentation: []string{"README.md"}}
	uc := usecase.NewInstallAssistantUseCase(fakeCatalogGateway{catalog: catalog}, installGateway, installGateway)

	result, err := uc.Execute(context.Background(), usecase.InstallRequest{
		Assistants:   []string{"instagram-post-studio"},
		AgentsPolicy: domain.DefaultAgentsPolicy,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(installGateway.assistantsInput) != 1 {
		t.Fatalf("expected 1 assistant selected, got %d", len(installGateway.assistantsInput))
	}

	if len(installGateway.skillsInput) != 2 {
		t.Fatalf("expected 2 associated skills selected, got %d", len(installGateway.skillsInput))
	}

	if installGateway.agentsTemplate != "/tmp/AGENTS.md" {
		t.Fatalf("expected AGENTS template path to be forwarded, got %q", installGateway.agentsTemplate)
	}

	if len(result.Installed) != 4 {
		t.Fatalf("expected merged installed length 4 (2 skills + 1 assistant + agents), got %d", len(result.Installed))
	}
}

func TestInstallAssistantUseCaseFailsOnUnknownAssistant(t *testing.T) {
	t.Parallel()

	catalog := usecase.Catalog{
		Assistants: []usecase.AssistantAsset{{ID: "assistant-a", SourcePath: "/tmp/assistant-a.yaml"}},
	}

	installGateway := &fakeInstallGateway{projectContext: domain.ProjectContext{Target: domain.TargetCodex, Title: "Heimdall", Description: "desc", Documentation: []string{"README.md"}}}
	uc := usecase.NewInstallAssistantUseCase(fakeCatalogGateway{catalog: catalog}, installGateway, installGateway)

	_, err := uc.Execute(context.Background(), usecase.InstallRequest{
		Assistants:   []string{"not-found"},
		AgentsPolicy: domain.DefaultAgentsPolicy,
	})
	if err == nil {
		t.Fatal("expected error for unknown assistant, got nil")
	}
	if !strings.Contains(err.Error(), "available assistants: [assistant-a]") {
		t.Fatalf("expected available assistants in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "available skills: none") {
		t.Fatalf("expected available skills in error, got %v", err)
	}
}

func TestInstallAssistantUseCaseFailsOnEmptyCatalog(t *testing.T) {
	t.Parallel()

	installGateway := &fakeInstallGateway{projectContext: domain.ProjectContext{Target: domain.TargetCodex, Title: "Heimdall", Description: "desc", Documentation: []string{"README.md"}}}
	uc := usecase.NewInstallAssistantUseCase(fakeCatalogGateway{catalog: usecase.Catalog{}}, installGateway, installGateway)

	_, err := uc.Execute(context.Background(), usecase.InstallRequest{
		Assistants:   []string{"assistant-a"},
		AgentsPolicy: domain.DefaultAgentsPolicy,
	})
	if err == nil {
		t.Fatal("expected error for empty catalog, got nil")
	}
	if !strings.Contains(err.Error(), "catalog does not contain installable items") {
		t.Fatalf("expected empty catalog message, got %v", err)
	}
}

func TestInstallAssistantUseCaseEmitsWarningForMissingAssociatedSkill(t *testing.T) {
	t.Parallel()

	catalog := usecase.Catalog{
		Skills: []usecase.SkillAsset{
			{Name: "designer", SourceDir: "/tmp/designer"},
		},
		Assistants: []usecase.AssistantAsset{
			{ID: "instagram-post-studio", SourcePath: "/tmp/instagram-post-studio.yaml", Skills: []string{"researcher", "designer"}},
		},
	}

	installGateway := &fakeInstallGateway{projectContext: domain.ProjectContext{Target: domain.TargetCodex, Title: "Heimdall", Description: "desc", Documentation: []string{"README.md"}}}
	uc := usecase.NewInstallAssistantUseCase(fakeCatalogGateway{catalog: catalog}, installGateway, installGateway)

	result, err := uc.Execute(context.Background(), usecase.InstallRequest{
		Assistants:   []string{"instagram-post-studio"},
		AgentsPolicy: domain.DefaultAgentsPolicy,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}
	if !strings.Contains(result.Warnings[0], `references missing skill "researcher"`) {
		t.Fatalf("expected missing skill warning, got %v", result.Warnings[0])
	}
}

func TestInstallAssistantUseCaseUsesTargetFromProjectContextWhenNotProvided(t *testing.T) {
	t.Parallel()

	catalog := usecase.Catalog{
		Assistants: []usecase.AssistantAsset{
			{ID: "assistant-a", SourcePath: "/tmp/assistant-a.yaml"},
		},
	}

	installGateway := &fakeInstallGateway{
		projectContext: domain.ProjectContext{
			Target:        domain.TargetClaude,
			ProjectRoot:   "/tmp/client-project",
			Title:         "Heimdall",
			Description:   "desc",
			Documentation: []string{"README.md"},
		},
	}
	uc := usecase.NewInstallAssistantUseCase(fakeCatalogGateway{catalog: catalog}, installGateway, installGateway)

	_, err := uc.Execute(context.Background(), usecase.InstallRequest{
		Assistants:   []string{"assistant-a"},
		AgentsPolicy: domain.DefaultAgentsPolicy,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if installGateway.lastRequest.Target != domain.TargetClaude {
		t.Fatalf("expected target resolved from project context, got %q", installGateway.lastRequest.Target)
	}
	if installGateway.lastRequest.OutputDir != "/tmp/client-project" {
		t.Fatalf("expected output dir resolved from project context project_root, got %q", installGateway.lastRequest.OutputDir)
	}
}

func TestInstallAssistantUseCaseInstallsAllWhenNoFilterProvided(t *testing.T) {
	t.Parallel()

	catalog := usecase.Catalog{
		Skills: []usecase.SkillAsset{
			{Name: "skill-a", SourceDir: "/tmp/skill-a"},
			{Name: "skill-b", SourceDir: "/tmp/skill-b"},
		},
		Assistants: []usecase.AssistantAsset{
			{ID: "assistant-a", SourcePath: "/tmp/assistant-a.yaml", Skills: []string{"skill-a"}},
			{ID: "assistant-b", SourcePath: "/tmp/assistant-b.yaml", Skills: []string{"skill-b"}},
		},
	}

	installGateway := &fakeInstallGateway{
		projectContext: domain.ProjectContext{Target: domain.TargetCodex, Title: "Heimdall", Description: "desc", Documentation: []string{"README.md"}},
	}
	uc := usecase.NewInstallAssistantUseCase(fakeCatalogGateway{catalog: catalog}, installGateway, installGateway)

	_, err := uc.Execute(context.Background(), usecase.InstallRequest{
		AgentsPolicy: domain.DefaultAgentsPolicy,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(installGateway.assistantsInput) != 2 {
		t.Fatalf("expected all assistants to be selected, got %d", len(installGateway.assistantsInput))
	}
}

func TestInstallAssistantUseCaseInstallsByCategory(t *testing.T) {
	t.Parallel()

	catalog := usecase.Catalog{
		Skills: []usecase.SkillAsset{
			{Name: "skill-a", SourceDir: "/tmp/skill-a"},
			{Name: "skill-b", SourceDir: "/tmp/skill-b"},
		},
		Assistants: []usecase.AssistantAsset{
			{ID: "assistant-a", SourcePath: "/tmp/assistant-a.yaml", Skills: []string{"skill-a"}, Categories: []string{"documentation"}},
			{ID: "assistant-b", SourcePath: "/tmp/assistant-b.yaml", Skills: []string{"skill-b"}, Categories: []string{"media"}},
		},
	}

	installGateway := &fakeInstallGateway{
		projectContext: domain.ProjectContext{Target: domain.TargetCodex, Title: "Heimdall", Description: "desc", Documentation: []string{"README.md"}},
	}
	uc := usecase.NewInstallAssistantUseCase(fakeCatalogGateway{catalog: catalog}, installGateway, installGateway)

	_, err := uc.Execute(context.Background(), usecase.InstallRequest{
		Category:     "documentation",
		AgentsPolicy: domain.DefaultAgentsPolicy,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(installGateway.assistantsInput) != 1 || installGateway.assistantsInput[0].ID != "assistant-a" {
		t.Fatalf("expected only assistant-a selected by category, got %#v", installGateway.assistantsInput)
	}
}

func TestInstallAssistantUseCaseInstallsSkillByID(t *testing.T) {
	t.Parallel()

	catalog := usecase.Catalog{
		Skills: []usecase.SkillAsset{
			{Name: "skill-a", SourceDir: "/tmp/skill-a"},
		},
		Assistants: []usecase.AssistantAsset{
			{ID: "assistant-a", SourcePath: "/tmp/assistant-a.yaml", Skills: []string{"skill-a"}},
		},
	}

	installGateway := &fakeInstallGateway{
		projectContext: domain.ProjectContext{Target: domain.TargetCodex, Title: "Heimdall", Description: "desc", Documentation: []string{"README.md"}},
	}
	uc := usecase.NewInstallAssistantUseCase(fakeCatalogGateway{catalog: catalog}, installGateway, installGateway)

	_, err := uc.Execute(context.Background(), usecase.InstallRequest{
		Assistants:   []string{"skill-a"},
		AgentsPolicy: domain.DefaultAgentsPolicy,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(installGateway.assistantsInput) != 0 {
		t.Fatalf("expected no assistants selected, got %#v", installGateway.assistantsInput)
	}
	if len(installGateway.skillsInput) != 1 || installGateway.skillsInput[0].Name != "skill-a" {
		t.Fatalf("expected direct skill installation for skill-a, got %#v", installGateway.skillsInput)
	}
}

func TestInstallAssistantUseCaseFailsWhenCategoryHasNoAssistants(t *testing.T) {
	t.Parallel()

	catalog := usecase.Catalog{
		Assistants: []usecase.AssistantAsset{
			{ID: "assistant-a", SourcePath: "/tmp/assistant-a.yaml", Categories: []string{"media"}},
		},
	}

	installGateway := &fakeInstallGateway{
		projectContext: domain.ProjectContext{Target: domain.TargetCodex, Title: "Heimdall", Description: "desc", Documentation: []string{"README.md"}},
	}
	uc := usecase.NewInstallAssistantUseCase(fakeCatalogGateway{catalog: catalog}, installGateway, installGateway)

	_, err := uc.Execute(context.Background(), usecase.InstallRequest{
		Category:     "documentation",
		AgentsPolicy: domain.DefaultAgentsPolicy,
	})
	if err == nil {
		t.Fatal("expected error when category has no assistants")
	}
}
