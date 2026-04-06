package template

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joleques/northstar-ai/src/domain"
	usecase "github.com/joleques/northstar-ai/src/use_case"
	"gopkg.in/yaml.v3"
)

type CatalogGateway struct {
	TemplateRoot string
}

func NewCatalogGateway(templateRoot string) CatalogGateway {
	return CatalogGateway{TemplateRoot: templateRoot}
}

func (g CatalogGateway) Load(_ context.Context, outputDir string) (usecase.Catalog, error) {
	clientRoot := resolveClientTemplateRoot(outputDir)
	if clientRoot != "" {
		catalog, err := g.loadFromRoot(clientRoot)
		if err != nil {
			return usecase.Catalog{}, err
		}
		if len(catalog.Skills) > 0 || len(catalog.Assistants) > 0 {
			return catalog, nil
		}
	}

	return g.loadFromRoot(g.TemplateRoot)
}

func (g CatalogGateway) loadFromRoot(root string) (usecase.Catalog, error) {
	templateRoot := strings.TrimSpace(root)
	if templateRoot == "" {
		return usecase.Catalog{}, fmt.Errorf("template root is required")
	}

	rootGateway := CatalogGateway{TemplateRoot: templateRoot}
	skills, assistants, err := rootGateway.loadTools()
	if err != nil {
		return usecase.Catalog{}, err
	}

	if len(skills) == 0 && len(assistants) == 0 {
		skills, err = rootGateway.loadLegacySkills()
		if err != nil {
			return usecase.Catalog{}, err
		}

		assistants, err = rootGateway.loadLegacyAssistants()
		if err != nil {
			return usecase.Catalog{}, err
		}
	}

	agentsTemplatePath := filepath.Join(templateRoot, "AGENTS.md")
	if _, err := os.Stat(agentsTemplatePath); err != nil {
		if !os.IsNotExist(err) {
			return usecase.Catalog{}, fmt.Errorf("read AGENTS template: %w", err)
		}

		legacyAgentsPath := filepath.Join(templateRoot, ".agent", "AGENTS.md")
		if _, legacyErr := os.Stat(legacyAgentsPath); legacyErr != nil {
			if !os.IsNotExist(legacyErr) {
				return usecase.Catalog{}, fmt.Errorf("read AGENTS template: %w", legacyErr)
			}
			agentsTemplatePath = ""
		} else {
			agentsTemplatePath = legacyAgentsPath
		}
	}

	return usecase.Catalog{
		Skills:             skills,
		Assistants:         assistants,
		AgentsTemplatePath: agentsTemplatePath,
	}, nil
}

func resolveClientTemplateRoot(outputDir string) string {
	base := strings.TrimSpace(outputDir)
	if base == "" {
		base = "."
	}

	root := filepath.Join(base, ".heimdall", "template")
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return ""
	}

	return root
}

func (g CatalogGateway) loadLegacySkills() ([]usecase.SkillAsset, error) {
	skillsDir := filepath.Join(g.TemplateRoot, ".agent", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []usecase.SkillAsset{}, nil
		}
		return nil, fmt.Errorf("read skills dir: %w", err)
	}

	skills := make([]usecase.SkillAsset, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(skillsDir, entry.Name())
		contract, err := loadSkillContract(skillDir)
		if err != nil {
			return nil, err
		}

		skills = append(skills, usecase.SkillAsset{
			Name:      entry.Name(),
			SourceDir: skillDir,
			Contract:  contract,
		})
	}

	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, nil
}

func (g CatalogGateway) loadLegacyAssistants() ([]usecase.AssistantAsset, error) {
	workflowsDir := filepath.Join(g.TemplateRoot, ".agent", "workflows")
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []usecase.AssistantAsset{}, nil
		}
		return nil, fmt.Errorf("read workflows dir: %w", err)
	}

	assistants := make([]usecase.AssistantAsset, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(workflowsDir, entry.Name())
		spec, err := parseAssistantYAML(path)
		if err != nil {
			return nil, err
		}

		assistants = append(assistants, usecase.AssistantAsset{
			ID:           spec.ID,
			Name:         spec.Name,
			Description:  spec.Description,
			Instructions: spec.Instructions,
			SourcePath:   path,
			Skills:       spec.Skills,
			Categories:   []string{},
		})
	}

	sort.Slice(assistants, func(i, j int) bool { return assistants[i].ID < assistants[j].ID })
	return assistants, nil
}

func (g CatalogGateway) loadTools() ([]usecase.SkillAsset, []usecase.AssistantAsset, error) {
	toolsDir := filepath.Join(g.TemplateRoot, "tools")
	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		if os.IsNotExist(err) {
			legacyToolsDir := filepath.Join(g.TemplateRoot, ".agent", "tools")
			legacyEntries, legacyErr := os.ReadDir(legacyToolsDir)
			if legacyErr != nil {
				if os.IsNotExist(legacyErr) {
					return []usecase.SkillAsset{}, []usecase.AssistantAsset{}, nil
				}
				return nil, nil, fmt.Errorf("read tools dir: %w", legacyErr)
			}
			toolsDir = legacyToolsDir
			entries = legacyEntries
		} else {
			return nil, nil, fmt.Errorf("read tools dir: %w", err)
		}
	}

	skills := make([]usecase.SkillAsset, 0)
	assistants := make([]usecase.AssistantAsset, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(toolsDir, entry.Name())
		rawType, err := parseToolType(path)
		if err != nil {
			return nil, nil, err
		}
		rawCategories, err := parseToolCategory(path)
		if err != nil {
			return nil, nil, err
		}
		normalizedCategories := make([]string, 0, len(rawCategories))
		for _, rawCategory := range rawCategories {
			normalizedCategory, err := normalizeToolCategory(rawCategory)
			if err != nil {
				return nil, nil, fmt.Errorf("validate tool %q: %w", path, err)
			}
			normalizedCategories = append(normalizedCategories, normalizedCategory)
		}
		normalizedCategories = dedupeCategories(normalizedCategories)

		switch normalizeToolType(rawType) {
		case "skill":
			contract, err := parseSkillContractYAML(path)
			if err != nil {
				return nil, nil, err
			}
			skillName := strings.TrimSpace(contract.Name)
			if skillName == "" {
				return nil, nil, fmt.Errorf("validate skill %q: name is required", path)
			}
			skills = append(skills, usecase.SkillAsset{
				Name:       skillName,
				Contract:   &contract,
				Categories: normalizedCategories,
			})
		case "assistant":
			spec, err := parseAssistantYAML(path)
			if err != nil {
				return nil, nil, err
			}
			assistants = append(assistants, usecase.AssistantAsset{
				ID:           spec.ID,
				Name:         spec.Name,
				Description:  spec.Description,
				Instructions: spec.Instructions,
				SourcePath:   path,
				Skills:       spec.Skills,
				Categories:   normalizedCategories,
			})
		default:
			return nil, nil, fmt.Errorf("validate tool %q: unsupported type %q (expected skill or assitent)", path, rawType)
		}
	}

	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	sort.Slice(assistants, func(i, j int) bool { return assistants[i].ID < assistants[j].ID })
	return skills, assistants, nil
}

type assistantYAML struct {
	Type         string             `yaml:"type"`
	Category     string             `yaml:"category"`
	Categories   []string           `yaml:"categories"`
	ID           string             `yaml:"id"`
	Name         string             `yaml:"name"`
	Description  string             `yaml:"description"`
	Instructions string             `yaml:"instructions"`
	Version      string             `yaml:"version"`
	Skills       []string           `yaml:"skills"`
	Inputs       []domain.InputSpec `yaml:"inputs"`
	Tools        []string           `yaml:"tools"`
	Tags         []string           `yaml:"tags"`
	Metadata     map[string]string  `yaml:"metadata"`
}

type skillYAML struct {
	Type         string   `yaml:"type"`
	Category     string   `yaml:"category"`
	Categories   []string `yaml:"categories"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Instructions string   `yaml:"instructions"`
}

type toolYAML struct {
	Type       string   `yaml:"type"`
	Category   string   `yaml:"category"`
	Categories []string `yaml:"categories"`
}

func loadSkillContract(skillDir string) (*usecase.SkillContract, error) {
	contractPath := filepath.Join(skillDir, "skill.yaml")
	if _, err := os.Stat(contractPath); err != nil {
		if os.IsNotExist(err) {
			contractPath = filepath.Join(skillDir, "skill.yml")
			if _, ymlErr := os.Stat(contractPath); ymlErr != nil {
				if os.IsNotExist(ymlErr) {
					return nil, nil
				}
				return nil, fmt.Errorf("read skill contract %q: %w", contractPath, ymlErr)
			}
		} else {
			return nil, fmt.Errorf("read skill contract %q: %w", contractPath, err)
		}
	}

	data, err := os.ReadFile(contractPath)
	if err != nil {
		return nil, fmt.Errorf("read skill contract %q: %w", contractPath, err)
	}

	var raw skillYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse skill contract YAML %q: %w", contractPath, err)
	}

	contract := usecase.SkillContract{
		Name:         strings.TrimSpace(raw.Name),
		Description:  strings.TrimSpace(raw.Description),
		Instructions: strings.TrimSpace(raw.Instructions),
	}

	if contract.Name == "" {
		return nil, fmt.Errorf("validate skill contract %q: name is required", contractPath)
	}
	if contract.Description == "" {
		return nil, fmt.Errorf("validate skill contract %q: description is required", contractPath)
	}
	if contract.Instructions == "" {
		return nil, fmt.Errorf("validate skill contract %q: instructions is required", contractPath)
	}

	return &contract, nil
}

func parseToolType(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read tool %q: %w", path, err)
	}

	var raw toolYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("parse tool YAML %q: %w", path, err)
	}

	toolType := strings.TrimSpace(raw.Type)
	if toolType == "" {
		return "", fmt.Errorf("validate tool %q: type is required", path)
	}
	return toolType, nil
}

func parseToolCategory(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tool %q: %w", path, err)
	}

	var raw toolYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse tool YAML %q: %w", path, err)
	}

	values := make([]string, 0, len(raw.Categories)+1)
	for _, category := range raw.Categories {
		clean := strings.TrimSpace(category)
		if clean == "" {
			continue
		}
		values = append(values, clean)
	}

	if len(values) == 0 {
		single := strings.TrimSpace(raw.Category)
		if single != "" {
			values = append(values, single)
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("validate tool %q: categories is required", path)
	}

	return values, nil
}

func parseSkillContractYAML(path string) (usecase.SkillContract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return usecase.SkillContract{}, fmt.Errorf("read skill contract %q: %w", path, err)
	}

	var raw skillYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return usecase.SkillContract{}, fmt.Errorf("parse skill contract YAML %q: %w", path, err)
	}

	contract := usecase.SkillContract{
		Name:         strings.TrimSpace(raw.Name),
		Description:  strings.TrimSpace(raw.Description),
		Instructions: strings.TrimSpace(raw.Instructions),
	}

	if contract.Name == "" {
		return usecase.SkillContract{}, fmt.Errorf("validate skill contract %q: name is required", path)
	}
	if contract.Description == "" {
		return usecase.SkillContract{}, fmt.Errorf("validate skill contract %q: description is required", path)
	}
	if contract.Instructions == "" {
		return usecase.SkillContract{}, fmt.Errorf("validate skill contract %q: instructions is required", path)
	}

	return contract, nil
}

func normalizeToolType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "assistant", "assitent":
		return "assistant"
	default:
		return normalized
	}
}

func normalizeToolCategory(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")

	switch normalized {
	case "software-architecture":
		return normalized, nil
	case "media":
		return normalized, nil
	case "documentation":
		return normalized, nil
	case "platform":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported category %q (expected software-architecture, media, documentation or platform)", value)
	}
}

func dedupeCategories(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func parseAssistantYAML(path string) (domain.AssistantSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.AssistantSpec{}, fmt.Errorf("read assistant %q: %w", path, err)
	}

	var raw assistantYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return domain.AssistantSpec{}, fmt.Errorf("parse assistant YAML %q: %w", path, err)
	}

	spec := domain.AssistantSpec{
		ID:           raw.ID,
		Name:         raw.Name,
		Description:  raw.Description,
		Instructions: raw.Instructions,
		Version:      raw.Version,
		Skills:       raw.Skills,
		Inputs:       raw.Inputs,
		Tools:        raw.Tools,
		Tags:         raw.Tags,
		Metadata:     raw.Metadata,
	}.Normalized()

	if err := spec.Validate(); err != nil {
		return domain.AssistantSpec{}, fmt.Errorf("validate assistant %q: %w", path, err)
	}

	return spec, nil
}
