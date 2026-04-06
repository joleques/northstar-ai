package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/joleques/northstar-ai/src/domain"
)

type SkillAsset struct {
	Name       string
	SourceDir  string
	Contract   *SkillContract
	Categories []string
}

type SkillContract struct {
	Name         string
	Description  string
	Instructions string
}

type AssistantAsset struct {
	ID           string
	Name         string
	Description  string
	Instructions string
	SourcePath   string
	Skills       []string
	Categories   []string
}

type Catalog struct {
	Skills             []SkillAsset
	Assistants         []AssistantAsset
	AgentsTemplatePath string
}

type CatalogGateway interface {
	Load(ctx context.Context, outputDir string) (Catalog, error)
}

type InstallGateway interface {
	InstallSkills(ctx context.Context, request InstallRequest, skills []SkillAsset) (InstallResult, error)
	InstallAssistants(ctx context.Context, request InstallRequest, assistants []AssistantAsset) (InstallResult, error)
	ApplyAgentsPolicy(ctx context.Context, request InstallRequest, templateAgentsPath string) (InstallResult, error)
}

type ProjectContextGateway interface {
	LoadProjectContext(ctx context.Context, outputDir string) (domain.ProjectContext, error)
}

type InstallAssistantUseCase struct {
	catalog        CatalogGateway
	install        InstallGateway
	projectContext ProjectContextGateway
}

func NewInstallAssistantUseCase(catalog CatalogGateway, install InstallGateway, projectContext ProjectContextGateway) InstallAssistantUseCase {
	return InstallAssistantUseCase{catalog: catalog, install: install, projectContext: projectContext}
}

func (uc InstallAssistantUseCase) Execute(ctx context.Context, request InstallRequest) (InstallResult, error) {
	if request.Target == "" {
		if uc.projectContext == nil {
			return InstallResult{}, fmt.Errorf("target is required when project context is unavailable")
		}

		projectContext, err := uc.projectContext.LoadProjectContext(ctx, request.OutputDir)
		if err != nil {
			return InstallResult{}, err
		}

		request.Target = projectContext.Target
		if strings.TrimSpace(request.OutputDir) == "" && strings.TrimSpace(projectContext.ProjectRoot) != "" {
			request.OutputDir = projectContext.ProjectRoot
		}
	}

	catalog, err := uc.catalog.Load(ctx, request.OutputDir)
	if err != nil {
		return InstallResult{}, err
	}

	if len(catalog.Assistants) == 0 && len(catalog.Skills) == 0 {
		return InstallResult{}, fmt.Errorf("catalog does not contain installable items")
	}

	selectedAssistants, selectedExplicitSkills, err := selectItemsForInstall(catalog, request)
	if err != nil {
		return InstallResult{}, err
	}

	selectedAssociatedSkills, warnings := selectSkillsForAssistants(catalog.Skills, selectedAssistants)
	selectedSkills := mergeSkills(selectedAssociatedSkills, selectedExplicitSkills)

	skillsResult, err := uc.install.InstallSkills(ctx, request, selectedSkills)
	if err != nil {
		return InstallResult{}, err
	}

	assistantResult, err := uc.install.InstallAssistants(ctx, request, selectedAssistants)
	if err != nil {
		return InstallResult{}, err
	}

	agentsResult, err := uc.install.ApplyAgentsPolicy(ctx, request, catalog.AgentsTemplatePath)
	if err != nil {
		return InstallResult{}, err
	}

	merged := mergeResults(skillsResult, assistantResult, agentsResult)
	merged.Warnings = append(merged.Warnings, warnings...)
	return merged, nil
}

func selectItemsForInstall(catalog Catalog, request InstallRequest) ([]AssistantAsset, []SkillAsset, error) {
	if len(request.Assistants) > 0 {
		return selectItemsByIDs(catalog.Assistants, catalog.Skills, request.Assistants)
	}

	if strings.TrimSpace(request.Category) != "" {
		filteredAssistants := filterAssistantsByCategory(catalog.Assistants, request.Category)
		filteredSkills := filterSkillsByCategory(catalog.Skills, request.Category)
		if len(filteredAssistants) == 0 && len(filteredSkills) == 0 {
			return nil, nil, fmt.Errorf("no installable items found for category %q", request.Category)
		}
		return filteredAssistants, filteredSkills, nil
	}

	allAssistants := append([]AssistantAsset(nil), catalog.Assistants...)
	allSkills := append([]SkillAsset(nil), catalog.Skills...)
	sort.Slice(allAssistants, func(i, j int) bool { return allAssistants[i].ID < allAssistants[j].ID })
	sort.Slice(allSkills, func(i, j int) bool { return allSkills[i].Name < allSkills[j].Name })
	return allAssistants, allSkills, nil
}

func selectAssistants(assistants []AssistantAsset, requested []string) ([]AssistantAsset, error) {
	byID := make(map[string]AssistantAsset, len(assistants))
	available := make([]string, 0, len(assistants))
	for _, assistant := range assistants {
		byID[assistant.ID] = assistant
		available = append(available, assistant.ID)
	}
	sort.Strings(available)

	selected := make([]AssistantAsset, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, id := range requested {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}

		assistant, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("assistant %q not found in catalog; available assistants: %s", id, formatAvailableAssistants(available))
		}
		selected = append(selected, assistant)
	}

	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	return selected, nil
}

func formatAvailableAssistants(available []string) string {
	if len(available) == 0 {
		return "none"
	}

	return fmt.Sprintf("[%s]", strings.Join(available, ", "))
}

func selectSkillsForAssistants(skills []SkillAsset, assistants []AssistantAsset) ([]SkillAsset, []string) {
	byName := make(map[string]SkillAsset, len(skills))
	for _, skill := range skills {
		byName[skill.Name] = skill
	}

	selected := make([]SkillAsset, 0)
	seen := make(map[string]struct{})
	warnings := make([]string, 0)

	for _, assistant := range assistants {
		for _, skillName := range assistant.Skills {
			if _, exists := seen[skillName]; exists {
				continue
			}
			seen[skillName] = struct{}{}

			skill, ok := byName[skillName]
			if !ok {
				warnings = append(warnings, fmt.Sprintf("assistant %q references missing skill %q", assistant.ID, skillName))
				continue
			}
			selected = append(selected, skill)
		}
	}

	sort.Slice(selected, func(i, j int) bool { return selected[i].Name < selected[j].Name })
	return selected, warnings
}

func filterAssistantsByCategory(assistants []AssistantAsset, category string) []AssistantAsset {
	desired := normalizeCategory(category)
	filtered := make([]AssistantAsset, 0)
	for _, assistant := range assistants {
		for _, current := range assistant.Categories {
			if normalizeCategory(current) == desired {
				filtered = append(filtered, assistant)
				break
			}
		}
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID < filtered[j].ID })
	return filtered
}

func filterSkillsByCategory(skills []SkillAsset, category string) []SkillAsset {
	desired := normalizeCategory(category)
	filtered := make([]SkillAsset, 0)
	for _, skill := range skills {
		for _, current := range skill.Categories {
			if normalizeCategory(current) == desired {
				filtered = append(filtered, skill)
				break
			}
		}
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	return filtered
}

func selectItemsByIDs(assistants []AssistantAsset, skills []SkillAsset, requested []string) ([]AssistantAsset, []SkillAsset, error) {
	assistantByID := make(map[string]AssistantAsset, len(assistants))
	assistantIDs := make([]string, 0, len(assistants))
	for _, assistant := range assistants {
		assistantByID[assistant.ID] = assistant
		assistantIDs = append(assistantIDs, assistant.ID)
	}
	sort.Strings(assistantIDs)

	skillByID := make(map[string]SkillAsset, len(skills))
	skillIDs := make([]string, 0, len(skills))
	for _, skill := range skills {
		skillByID[skill.Name] = skill
		skillIDs = append(skillIDs, skill.Name)
	}
	sort.Strings(skillIDs)

	selectedAssistants := make([]AssistantAsset, 0, len(requested))
	selectedSkills := make([]SkillAsset, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))

	for _, id := range requested {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}

		matched := false
		if assistant, ok := assistantByID[id]; ok {
			selectedAssistants = append(selectedAssistants, assistant)
			matched = true
		}
		if skill, ok := skillByID[id]; ok {
			selectedSkills = append(selectedSkills, skill)
			matched = true
		}
		if !matched {
			return nil, nil, fmt.Errorf(
				"item %q not found in catalog; available assistants: %s; available skills: %s",
				id,
				formatAvailableAssistants(assistantIDs),
				formatAvailableAssistants(skillIDs),
			)
		}
	}

	sort.Slice(selectedAssistants, func(i, j int) bool { return selectedAssistants[i].ID < selectedAssistants[j].ID })
	sort.Slice(selectedSkills, func(i, j int) bool { return selectedSkills[i].Name < selectedSkills[j].Name })
	return selectedAssistants, selectedSkills, nil
}

func mergeSkills(primary []SkillAsset, additional []SkillAsset) []SkillAsset {
	byName := make(map[string]SkillAsset, len(primary)+len(additional))
	for _, skill := range primary {
		byName[skill.Name] = skill
	}
	for _, skill := range additional {
		byName[skill.Name] = skill
	}

	merged := make([]SkillAsset, 0, len(byName))
	for _, skill := range byName {
		merged = append(merged, skill)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Name < merged[j].Name })
	return merged
}

func normalizeCategory(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	return normalized
}

func mergeResults(results ...InstallResult) InstallResult {
	merged := InstallResult{
		Installed: make([]string, 0),
		Skipped:   make([]string, 0),
		Failed:    make([]string, 0),
		Warnings:  make([]string, 0),
	}

	for _, result := range results {
		merged.Installed = append(merged.Installed, result.Installed...)
		merged.Skipped = append(merged.Skipped, result.Skipped...)
		merged.Failed = append(merged.Failed, result.Failed...)
		merged.Warnings = append(merged.Warnings, result.Warnings...)
	}

	return merged
}

var _ InstallAssistant = InstallAssistantUseCase{}
