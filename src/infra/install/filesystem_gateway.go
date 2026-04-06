package install

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/heimdall-app/heimdall/src/domain"
	usecase "github.com/heimdall-app/heimdall/src/use_case"
	"gopkg.in/yaml.v3"
)

type FilesystemGateway struct {
	agentsTemplatePath string
	templateSourceDir  string
}

type managedToolsState struct {
	Skills     []string `yaml:"skills"`
	Assistants []string `yaml:"assistants"`
}

func NewFilesystemGateway(agentsTemplatePath ...string) FilesystemGateway {
	path := ""
	if len(agentsTemplatePath) > 0 {
		path = agentsTemplatePath[0]
	}

	templateSourceDir := ""
	if len(agentsTemplatePath) > 1 {
		templateSourceDir = agentsTemplatePath[1]
	}
	if strings.TrimSpace(templateSourceDir) == "" {
		templateSourceDir = resolveTemplateSourceDir(path)
	}

	return FilesystemGateway{
		agentsTemplatePath: path,
		templateSourceDir:  templateSourceDir,
	}
}

func (g FilesystemGateway) LoadProjectContext(_ context.Context, outputDir string) (domain.ProjectContext, error) {
	layout := resolveProjectContextLayout(outputDir)

	data, err := os.ReadFile(layout.ManifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.ProjectContext{}, fmt.Errorf("project context not found at %q; run 'heimdall start' first", layout.ManifestPath)
		}
		return domain.ProjectContext{}, fmt.Errorf("read project context manifest: %w", err)
	}

	var manifest projectContextManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return domain.ProjectContext{}, fmt.Errorf("parse project context manifest: %w", err)
	}

	documentation := make([]string, 0, len(manifest.Documents))
	for _, document := range manifest.Documents {
		documentation = append(documentation, document.Source)
	}

	context := domain.ProjectContext{
		Target:        manifest.Target,
		ProjectRoot:   manifest.ProjectRoot,
		Title:         manifest.Title,
		Description:   manifest.Description,
		Documentation: documentation,
	}.Normalized()
	if context.ProjectRoot == "" {
		context.ProjectRoot = resolveProjectRoot(outputDir)
	}

	if _, err := domain.ParseTargetPlatform(string(context.Target)); err != nil {
		return domain.ProjectContext{}, fmt.Errorf("invalid project context: target is required")
	}

	return context, nil
}

func (g FilesystemGateway) SaveProjectContext(_ context.Context, request usecase.StartRequest) (usecase.StartResult, error) {
	layout := resolveProjectContextLayout(request.OutputDir)

	result := usecase.StartResult{
		Created:  make([]string, 0, len(request.Documentation)+4),
		Updated:  make([]string, 0, 1),
		Skipped:  make([]string, 0, len(request.Documentation)+4),
		Warnings: make([]string, 0),
	}

	for _, dir := range []string{layout.RootDir, layout.DocsDir} {
		created, err := createDirWithIdempotency(dir, request.Force)
		if err != nil {
			return usecase.StartResult{}, err
		}
		if created {
			result.Created = append(result.Created, dir)
		} else {
			result.Skipped = append(result.Skipped, dir)
		}
	}

	documents := make([]projectContextDocument, 0, len(request.Documentation))
	for index, entry := range request.Documentation {
		document, output, err := persistDocumentationEntry(layout, index, entry, request.Force)
		if err != nil {
			return usecase.StartResult{}, err
		}
		documents = append(documents, document)
		appendFileOutcome(&result, output)
	}

	manifestBytes, err := yaml.Marshal(projectContextManifest{
		Target:      request.Target,
		ProjectRoot: resolveProjectRoot(request.OutputDir),
		Title:       request.Title,
		Description: request.Description,
		Documents:   documents,
	})
	if err != nil {
		return usecase.StartResult{}, fmt.Errorf("marshal project context manifest: %w", err)
	}

	manifestOutput, err := writeFileWithOverwrite(layout.ManifestPath, manifestBytes)
	if err != nil {
		return usecase.StartResult{}, err
	}
	appendOverwrittenFileOutcome(&result, manifestOutput)

	return result, nil
}

func (g FilesystemGateway) InitTarget(ctx context.Context, request usecase.InitRequest) (usecase.InitResult, error) {
	layout, err := resolveLayout(request.Target, request.OutputDir)
	if err != nil {
		return usecase.InitResult{}, err
	}

	result := usecase.InitResult{
		Created:  make([]string, 0, 4),
		Skipped:  make([]string, 0, 4),
		Warnings: make([]string, 0),
	}

	for _, dir := range []string{layout.SkillsDir} {
		created, err := createDirWithIdempotency(dir, request.Force)
		if err != nil {
			return usecase.InitResult{}, err
		}
		if created {
			result.Created = append(result.Created, dir)
		} else {
			result.Skipped = append(result.Skipped, dir)
		}
	}

	contextOutput, err := g.ensureProjectContextTarget(request)
	if err != nil {
		return usecase.InitResult{}, err
	}
	appendInitFileOutcome(&result, contextOutput)

	gitignoreChanged, gitignorePath, err := ensureGitignoreHasHeimdall(request.OutputDir)
	if err != nil {
		return usecase.InitResult{}, err
	}
	if gitignoreChanged {
		result.Created = append(result.Created, gitignorePath)
	} else {
		result.Skipped = append(result.Skipped, gitignorePath)
	}

	templateOutput, templateErr := g.copyTemplateSnapshot(request.OutputDir, request.Force)
	if templateErr != nil {
		return usecase.InitResult{}, templateErr
	}
	if templateOutput.Path != "" {
		if templateOutput.Created {
			result.Created = append(result.Created, templateOutput.Path)
		} else {
			result.Skipped = append(result.Skipped, templateOutput.Path)
		}
	}

	platformToolsResult, err := g.installPlatformTools(ctx, request)
	if err != nil {
		return usecase.InitResult{}, err
	}
	result.Created = append(result.Created, platformToolsResult.Installed...)
	result.Skipped = append(result.Skipped, platformToolsResult.Skipped...)
	result.Warnings = append(result.Warnings, platformToolsResult.Warnings...)
	for _, failed := range platformToolsResult.Failed {
		result.Warnings = append(result.Warnings, "platform-tool:"+failed)
	}

	agentsResult, err := g.ApplyAgentsPolicy(ctx, usecase.InstallRequest{
		Target:       request.Target,
		AgentsPolicy: request.AgentsPolicy,
		Force:        request.Force,
		OutputDir:    request.OutputDir,
	}, g.agentsTemplatePath)
	if err != nil {
		return usecase.InitResult{}, err
	}

	result.Created = append(result.Created, agentsResult.Installed...)
	result.Skipped = append(result.Skipped, agentsResult.Skipped...)
	result.Warnings = append(result.Warnings, agentsResult.Warnings...)

	return result, nil
}

func (g FilesystemGateway) UpdateApp(ctx context.Context, request usecase.UpdateAppRequest) (usecase.UpdateAppResult, error) {
	layout, err := resolveLayout(request.Target, request.OutputDir)
	if err != nil {
		return usecase.UpdateAppResult{}, err
	}

	sourceDir := strings.TrimSpace(g.templateSourceDir)
	if sourceDir == "" {
		return usecase.UpdateAppResult{}, fmt.Errorf("template source unavailable for update-app")
	}

	currentCatalog, err := g.loadTemplateCatalogFromToolsDir(filepath.Join(sourceDir, "tools"))
	if err != nil {
		return usecase.UpdateAppResult{}, err
	}

	currentPlatformTools, err := g.loadPlatformToolsFromToolsDir(filepath.Join(sourceDir, "tools"))
	if err != nil {
		return usecase.UpdateAppResult{}, err
	}
	currentPlatformSkills := platformToolsToSkillAssets(currentPlatformTools)

	previousPlatformTools, err := g.loadPlatformToolsFromToolsDir(filepath.Join(resolveHeimdallLayout(request.OutputDir).TemplateDir, "tools"))
	if err != nil {
		return usecase.UpdateAppResult{}, err
	}
	previousPlatformSkills := platformToolsToSkillAssets(previousPlatformTools)

	state, err := loadManagedToolsState(request.OutputDir)
	if err != nil {
		return usecase.UpdateAppResult{}, err
	}

	result := usecase.UpdateAppResult{
		Removed:   []string{},
		Installed: []string{},
		Skipped:   []string{},
		Failed:    []string{},
		Warnings:  []string{},
	}

	platformSkillNames := make(map[string]struct{}, len(currentPlatformSkills)+len(previousPlatformSkills))
	for _, skill := range currentPlatformSkills {
		platformSkillNames[skill.Name] = struct{}{}
	}
	for _, skill := range previousPlatformSkills {
		platformSkillNames[skill.Name] = struct{}{}
	}

	for skillName := range platformSkillNames {
		path := filepath.Join(layout.SkillsDir, skillName)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				result.Skipped = append(result.Skipped, "skill:"+skillName)
				continue
			}

			result.Failed = append(result.Failed, "skill:"+skillName)
			result.Warnings = append(result.Warnings, fmt.Sprintf("stat platform skill %q: %v", path, err))
			continue
		}

		if err := os.RemoveAll(path); err != nil {
			result.Failed = append(result.Failed, "skill:"+skillName)
			result.Warnings = append(result.Warnings, fmt.Sprintf("remove platform skill %q: %v", path, err))
			continue
		}

		result.Removed = append(result.Removed, "skill:"+skillName)
	}

	templateToolsResult, err := g.refreshPlatformToolsSnapshot(request.OutputDir, currentPlatformTools, previousPlatformTools)
	if err != nil {
		return usecase.UpdateAppResult{}, err
	}
	result.Removed = append(result.Removed, templateToolsResult.Removed...)
	result.Installed = append(result.Installed, templateToolsResult.Installed...)
	result.Failed = append(result.Failed, templateToolsResult.Failed...)
	result.Warnings = append(result.Warnings, templateToolsResult.Warnings...)

	managedSkillNames := make(map[string]struct{}, len(state.Skills)+len(currentPlatformSkills))
	for _, skill := range state.Skills {
		normalized := strings.TrimSpace(skill)
		if normalized != "" {
			managedSkillNames[normalized] = struct{}{}
		}
	}
	for _, skill := range currentPlatformSkills {
		if skill.Name != "" {
			managedSkillNames[skill.Name] = struct{}{}
		}
	}

	skillsToInstall := make([]usecase.SkillAsset, 0, len(managedSkillNames))
	missingManagedSkills := make([]string, 0)
	for skillName := range managedSkillNames {
		if skill, exists := currentCatalog.SkillsByName[skillName]; exists {
			skillsToInstall = append(skillsToInstall, skill)
		} else {
			missingManagedSkills = append(missingManagedSkills, skillName)
		}
	}

	skillsResult, err := g.InstallSkills(ctx, usecase.InstallRequest{
		Target:       request.Target,
		Force:        true,
		OutputDir:    request.OutputDir,
		SkipWrappers: true,
	}, skillsToInstall)
	if err != nil {
		return usecase.UpdateAppResult{}, err
	}

	managedAssistants := make([]usecase.AssistantAsset, 0, len(state.Assistants))
	missingManagedAssistants := make([]string, 0)
	for _, assistantID := range state.Assistants {
		normalized := strings.TrimSpace(assistantID)
		if normalized == "" {
			continue
		}

		if assistant, exists := currentCatalog.AssistantsByID[normalized]; exists {
			managedAssistants = append(managedAssistants, assistant)
		} else {
			missingManagedAssistants = append(missingManagedAssistants, normalized)
		}
	}

	assistantsResult, err := g.InstallAssistants(ctx, usecase.InstallRequest{
		Target:    request.Target,
		Force:     true,
		OutputDir: request.OutputDir,
	}, managedAssistants)
	if err != nil {
		return usecase.UpdateAppResult{}, err
	}

	result.Installed = append(result.Installed, skillsResult.Installed...)
	result.Skipped = append(result.Skipped, skillsResult.Skipped...)
	result.Failed = append(result.Failed, skillsResult.Failed...)
	result.Warnings = append(result.Warnings, skillsResult.Warnings...)

	result.Installed = append(result.Installed, assistantsResult.Installed...)
	result.Skipped = append(result.Skipped, assistantsResult.Skipped...)
	result.Failed = append(result.Failed, assistantsResult.Failed...)
	result.Warnings = append(result.Warnings, assistantsResult.Warnings...)

	removedSkills, removedSkillWarnings := removeManagedSkillsFromDisk(layout, missingManagedSkills)
	result.Removed = append(result.Removed, removedSkills...)
	result.Warnings = append(result.Warnings, removedSkillWarnings...)

	removedAssistants, removedAssistantWarnings := removeManagedAssistantsFromDisk(layout, request.Target, missingManagedAssistants)
	result.Removed = append(result.Removed, removedAssistants...)
	result.Warnings = append(result.Warnings, removedAssistantWarnings...)

	if err := removeManagedTools(request.OutputDir, missingManagedSkills, missingManagedAssistants); err != nil {
		return usecase.UpdateAppResult{}, err
	}

	return result, nil
}

func (g FilesystemGateway) ensureProjectContextTarget(request usecase.InitRequest) (fileWriteOutput, error) {
	layout := resolveProjectContextLayout(request.OutputDir)
	created, err := createDirWithIdempotency(layout.RootDir, request.Force)
	if err != nil {
		return fileWriteOutput{}, err
	}

	if !created && !request.Force {
		if _, statErr := os.Stat(layout.ManifestPath); statErr == nil {
			return fileWriteOutput{Path: layout.ManifestPath, Created: false}, nil
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return fileWriteOutput{}, fmt.Errorf("stat project context manifest %q: %w", layout.ManifestPath, statErr)
		}
	}

	manifestBytes, err := yaml.Marshal(projectContextManifest{
		Target:      request.Target,
		ProjectRoot: resolveProjectRoot(request.OutputDir),
		Documents:   []projectContextDocument{},
	})
	if err != nil {
		return fileWriteOutput{}, fmt.Errorf("marshal project context manifest: %w", err)
	}

	output, err := writeFileWithIdempotency(layout.ManifestPath, manifestBytes, request.Force)
	if err != nil {
		return fileWriteOutput{}, err
	}
	return output, nil
}

func (g FilesystemGateway) installPlatformTools(ctx context.Context, request usecase.InitRequest) (usecase.InstallResult, error) {
	sourceDir := strings.TrimSpace(g.templateSourceDir)
	if sourceDir == "" {
		return usecase.InstallResult{}, nil
	}

	skills, err := g.loadPlatformSkillsFromToolsDir(filepath.Join(sourceDir, "tools"))
	if err != nil {
		return usecase.InstallResult{}, err
	}

	installRequest := usecase.InstallRequest{
		Target:       request.Target,
		Force:        request.Force,
		OutputDir:    request.OutputDir,
		SkipWrappers: true,
	}

	skillsResult, err := g.InstallSkills(ctx, installRequest, skills)
	if err != nil {
		return usecase.InstallResult{}, err
	}

	return skillsResult, nil
}

func (g FilesystemGateway) loadPlatformSkillsFromToolsDir(toolsDir string) ([]usecase.SkillAsset, error) {
	tools, err := g.loadPlatformToolsFromToolsDir(toolsDir)
	if err != nil {
		return nil, err
	}
	return platformToolsToSkillAssets(tools), nil
}

type platformToolAsset struct {
	FileName string
	Source   string
	Skill    usecase.SkillAsset
}

type templateToolCatalog struct {
	SkillsByName   map[string]usecase.SkillAsset
	AssistantsByID map[string]usecase.AssistantAsset
}

func (g FilesystemGateway) loadPlatformToolsFromToolsDir(toolsDir string) ([]platformToolAsset, error) {
	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []platformToolAsset{}, nil
		}
		return nil, fmt.Errorf("read tools dir for platform auto-install: %w", err)
	}

	tools := make([]platformToolAsset, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		toolPath := filepath.Join(toolsDir, entry.Name())
		tool, err := parseTemplateToolYAML(toolPath)
		if err != nil {
			return nil, err
		}

		if !hasCategory(tool.Categories, "platform") {
			continue
		}

		switch normalizeTemplateToolType(tool.Type) {
		case "skill":
			skillName := strings.TrimSpace(tool.Name)
			if skillName == "" {
				return nil, fmt.Errorf("validate platform tool %q: skill name is required", toolPath)
			}

			tools = append(tools, platformToolAsset{
				FileName: entry.Name(),
				Source:   toolPath,
				Skill: usecase.SkillAsset{
					Name: skillName,
					Contract: &usecase.SkillContract{
						Name:         skillName,
						Description:  strings.TrimSpace(tool.Description),
						Instructions: strings.TrimSpace(tool.Instructions),
					},
				},
			})
		case "assistant":
			assistantID := strings.TrimSpace(tool.ID)
			if assistantID == "" {
				return nil, fmt.Errorf("validate platform tool %q: assistant id is required", toolPath)
			}

			assistantName := strings.TrimSpace(tool.Name)
			if assistantName == "" {
				assistantName = assistantID
			}

			tools = append(tools, platformToolAsset{
				FileName: entry.Name(),
				Source:   toolPath,
				Skill: usecase.SkillAsset{
					Name: assistantID,
					Contract: &usecase.SkillContract{
						Name:         assistantName,
						Description:  strings.TrimSpace(tool.Description),
						Instructions: buildPlatformAssistantSkillInstructions(tool.Instructions, tool.Skills),
					},
				},
			})
		default:
			return nil, fmt.Errorf("validate platform tool %q: unsupported type %q", toolPath, tool.Type)
		}
	}

	return tools, nil
}

func platformToolsToSkillAssets(tools []platformToolAsset) []usecase.SkillAsset {
	skills := make([]usecase.SkillAsset, 0, len(tools))
	for _, tool := range tools {
		skills = append(skills, tool.Skill)
	}
	return skills
}

func (g FilesystemGateway) refreshPlatformToolsSnapshot(outputDir string, currentTools []platformToolAsset, previousTools []platformToolAsset) (usecase.UpdateAppResult, error) {
	result := usecase.UpdateAppResult{
		Removed:   []string{},
		Installed: []string{},
		Skipped:   []string{},
		Failed:    []string{},
		Warnings:  []string{},
	}

	toolsDir := filepath.Join(resolveHeimdallLayout(outputDir).TemplateDir, "tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return usecase.UpdateAppResult{}, fmt.Errorf("create tools snapshot dir %q: %w", toolsDir, err)
	}

	currentByFile := make(map[string]platformToolAsset, len(currentTools))
	for _, tool := range currentTools {
		currentByFile[tool.FileName] = tool
	}

	for _, tool := range previousTools {
		if _, exists := currentByFile[tool.FileName]; exists {
			continue
		}

		destination := filepath.Join(toolsDir, tool.FileName)
		if _, err := os.Stat(destination); err != nil {
			if os.IsNotExist(err) {
				result.Skipped = append(result.Skipped, "template-tool:"+tool.FileName)
				continue
			}
			result.Failed = append(result.Failed, "template-tool:"+tool.FileName)
			result.Warnings = append(result.Warnings, fmt.Sprintf("stat template tool %q: %v", destination, err))
			continue
		}

		if err := os.Remove(destination); err != nil {
			result.Failed = append(result.Failed, "template-tool:"+tool.FileName)
			result.Warnings = append(result.Warnings, fmt.Sprintf("remove template tool %q: %v", destination, err))
			continue
		}
		result.Removed = append(result.Removed, "template-tool:"+tool.FileName)
	}

	for _, tool := range currentTools {
		destination := filepath.Join(toolsDir, tool.FileName)
		copied, err := copyFileWithIdempotency(tool.Source, destination, true)
		if err != nil {
			result.Failed = append(result.Failed, "template-tool:"+tool.FileName)
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}

		if copied {
			result.Installed = append(result.Installed, "template-tool:"+tool.FileName)
			continue
		}
		result.Skipped = append(result.Skipped, "template-tool:"+tool.FileName)
	}

	return result, nil
}

func (g FilesystemGateway) loadTemplateCatalogFromToolsDir(toolsDir string) (templateToolCatalog, error) {
	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return templateToolCatalog{
				SkillsByName:   map[string]usecase.SkillAsset{},
				AssistantsByID: map[string]usecase.AssistantAsset{},
			}, nil
		}
		return templateToolCatalog{}, fmt.Errorf("read tools dir for template catalog: %w", err)
	}

	catalog := templateToolCatalog{
		SkillsByName:   make(map[string]usecase.SkillAsset),
		AssistantsByID: make(map[string]usecase.AssistantAsset),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		toolPath := filepath.Join(toolsDir, entry.Name())
		tool, err := parseTemplateToolYAML(toolPath)
		if err != nil {
			return templateToolCatalog{}, err
		}

		switch normalizeTemplateToolType(tool.Type) {
		case "skill":
			skillName := strings.TrimSpace(tool.Name)
			if skillName == "" {
				return templateToolCatalog{}, fmt.Errorf("validate tool %q: skill name is required", toolPath)
			}

			catalog.SkillsByName[skillName] = usecase.SkillAsset{
				Name: skillName,
				Contract: &usecase.SkillContract{
					Name:         skillName,
					Description:  strings.TrimSpace(tool.Description),
					Instructions: strings.TrimSpace(tool.Instructions),
				},
			}
		case "assistant":
			assistantID := strings.TrimSpace(tool.ID)
			if assistantID == "" {
				return templateToolCatalog{}, fmt.Errorf("validate tool %q: assistant id is required", toolPath)
			}

			catalog.AssistantsByID[assistantID] = usecase.AssistantAsset{
				ID:           assistantID,
				Name:         strings.TrimSpace(tool.Name),
				Description:  strings.TrimSpace(tool.Description),
				Instructions: strings.TrimSpace(tool.Instructions),
				SourcePath:   toolPath,
				Skills:       dedupeAndTrim(tool.Skills),
				Categories:   dedupeAndTrim(tool.Categories),
			}
		default:
			return templateToolCatalog{}, fmt.Errorf("validate tool %q: unsupported type %q", toolPath, tool.Type)
		}
	}

	return catalog, nil
}

func managedToolsStatePath(outputDir string) string {
	heimdall := resolveHeimdallLayout(outputDir)
	return filepath.Join(heimdall.RootDir, "state", "managed-tools.yaml")
}

func loadManagedToolsState(outputDir string) (managedToolsState, error) {
	path := managedToolsStatePath(outputDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return managedToolsState{
				Skills:     []string{},
				Assistants: []string{},
			}, nil
		}
		return managedToolsState{}, fmt.Errorf("read managed tools state %q: %w", path, err)
	}

	var state managedToolsState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return managedToolsState{}, fmt.Errorf("parse managed tools state %q: %w", path, err)
	}

	state.Skills = dedupeAndTrim(state.Skills)
	state.Assistants = dedupeAndTrim(state.Assistants)
	return state, nil
}

func saveManagedToolsState(outputDir string, state managedToolsState) error {
	state.Skills = dedupeAndTrim(state.Skills)
	state.Assistants = dedupeAndTrim(state.Assistants)

	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal managed tools state: %w", err)
	}

	path := managedToolsStatePath(outputDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create managed tools state dir for %q: %w", path, err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write managed tools state %q: %w", path, err)
	}

	return nil
}

func addManagedSkills(outputDir string, names []string) error {
	if len(names) == 0 {
		return nil
	}

	state, err := loadManagedToolsState(outputDir)
	if err != nil {
		return err
	}

	state.Skills = append(state.Skills, names...)
	return saveManagedToolsState(outputDir, state)
}

func addManagedAssistants(outputDir string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	state, err := loadManagedToolsState(outputDir)
	if err != nil {
		return err
	}

	state.Assistants = append(state.Assistants, ids...)
	return saveManagedToolsState(outputDir, state)
}

func removeManagedTools(outputDir string, skillNames []string, assistantIDs []string) error {
	if len(skillNames) == 0 && len(assistantIDs) == 0 {
		return nil
	}

	state, err := loadManagedToolsState(outputDir)
	if err != nil {
		return err
	}

	removeSkills := make(map[string]struct{}, len(skillNames))
	for _, skill := range skillNames {
		normalized := strings.TrimSpace(skill)
		if normalized != "" {
			removeSkills[normalized] = struct{}{}
		}
	}

	removeAssistants := make(map[string]struct{}, len(assistantIDs))
	for _, assistant := range assistantIDs {
		normalized := strings.TrimSpace(assistant)
		if normalized != "" {
			removeAssistants[normalized] = struct{}{}
		}
	}

	filteredSkills := make([]string, 0, len(state.Skills))
	for _, skill := range state.Skills {
		if _, remove := removeSkills[skill]; remove {
			continue
		}
		filteredSkills = append(filteredSkills, skill)
	}
	state.Skills = filteredSkills

	filteredAssistants := make([]string, 0, len(state.Assistants))
	for _, assistant := range state.Assistants {
		if _, remove := removeAssistants[assistant]; remove {
			continue
		}
		filteredAssistants = append(filteredAssistants, assistant)
	}
	state.Assistants = filteredAssistants

	return saveManagedToolsState(outputDir, state)
}

func removeManagedSkillsFromDisk(layout targetLayout, skillNames []string) ([]string, []string) {
	removed := make([]string, 0, len(skillNames))
	warnings := make([]string, 0)

	for _, skillName := range skillNames {
		normalized := strings.TrimSpace(skillName)
		if normalized == "" {
			continue
		}

		path := filepath.Join(layout.SkillsDir, normalized)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			warnings = append(warnings, fmt.Sprintf("stat managed skill %q: %v", path, err))
			continue
		}

		if err := os.RemoveAll(path); err != nil {
			warnings = append(warnings, fmt.Sprintf("remove managed skill %q: %v", path, err))
			continue
		}

		removed = append(removed, "skill:"+normalized)
	}

	return removed, warnings
}

func removeManagedAssistantsFromDisk(layout targetLayout, _ domain.TargetPlatform, assistantIDs []string) ([]string, []string) {
	removed := make([]string, 0, len(assistantIDs))
	warnings := make([]string, 0)

	for _, assistantID := range assistantIDs {
		normalized := strings.TrimSpace(assistantID)
		if normalized == "" {
			continue
		}

		removedAssistant := false
		for _, ext := range []string{".yaml", ".yml"} {
			path := filepath.Join(layout.AssistantsDir, normalized+ext)
			if _, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				warnings = append(warnings, fmt.Sprintf("stat managed assistant %q: %v", path, err))
				continue
			}

			if err := os.Remove(path); err != nil {
				warnings = append(warnings, fmt.Sprintf("remove managed assistant %q: %v", path, err))
				continue
			}
			removedAssistant = true
		}

		wrapperPath := filepath.Join(layout.SkillsDir, "assistant-"+normalized)
		if _, err := os.Stat(wrapperPath); err == nil {
			if removeErr := os.RemoveAll(wrapperPath); removeErr != nil {
				warnings = append(warnings, fmt.Sprintf("remove managed assistant wrapper %q: %v", wrapperPath, removeErr))
			} else {
				removed = append(removed, "assistant-wrapper:"+normalized)
				removedAssistant = true
			}
		}

		if removedAssistant {
			removed = append(removed, "assistant:"+normalized)
		}
	}

	return removed, warnings
}

func (g FilesystemGateway) InstallSkills(_ context.Context, request usecase.InstallRequest, skills []usecase.SkillAsset) (usecase.InstallResult, error) {
	layout, err := resolveLayout(request.Target, request.OutputDir)
	if err != nil {
		return usecase.InstallResult{}, err
	}

	if err := os.MkdirAll(layout.SkillsDir, 0o755); err != nil {
		return usecase.InstallResult{}, fmt.Errorf("create skills dir: %w", err)
	}

	result := usecase.InstallResult{}
	managedInstalled := make([]string, 0, len(skills))
	for _, skill := range skills {
		dest := filepath.Join(layout.SkillsDir, skill.Name)
		copied := false
		if strings.TrimSpace(skill.SourceDir) != "" {
			var err error
			copied, err = copyDirWithIdempotency(skill.SourceDir, dest, request.Force)
			if err != nil {
				result.Failed = append(result.Failed, "skill:"+skill.Name)
				result.Warnings = append(result.Warnings, err.Error())
				continue
			}
		} else {
			var err error
			copied, err = createDirWithIdempotency(dest, request.Force)
			if err != nil {
				result.Failed = append(result.Failed, "skill:"+skill.Name)
				result.Warnings = append(result.Warnings, err.Error())
				continue
			}
		}

		if err := ensureSkillMarkdown(dest, skill, request.Force); err != nil {
			result.Failed = append(result.Failed, "skill:"+skill.Name)
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}

		if copied {
			result.Installed = append(result.Installed, "skill:"+skill.Name)
			managedInstalled = append(managedInstalled, skill.Name)
		} else {
			result.Skipped = append(result.Skipped, "skill:"+skill.Name)
		}
	}

	if err := addManagedSkills(request.OutputDir, managedInstalled); err != nil {
		return usecase.InstallResult{}, err
	}

	return result, nil
}

func (g FilesystemGateway) InstallAssistants(_ context.Context, request usecase.InstallRequest, assistants []usecase.AssistantAsset) (usecase.InstallResult, error) {
	layout, err := resolveLayout(request.Target, request.OutputDir)
	if err != nil {
		return usecase.InstallResult{}, err
	}

	if err := os.MkdirAll(layout.SkillsDir, 0o755); err != nil {
		return usecase.InstallResult{}, fmt.Errorf("create skills dir for assistants: %w", err)
	}

	result := usecase.InstallResult{}
	managedInstalled := make([]string, 0, len(assistants))
	for _, assistant := range assistants {
		if request.SkipWrappers {
			result.Skipped = append(result.Skipped, "assistant:"+assistant.ID)
			continue
		}

		wrapperOutput, wrapperErr := installAssistantWrapper(layout, request.Force, assistant)
		if wrapperErr != nil {
			result.Failed = append(result.Failed, "assistant:"+assistant.ID)
			result.Warnings = append(result.Warnings, wrapperErr.Error())
			continue
		}

		if wrapperOutput.Created {
			result.Installed = append(result.Installed, "assistant:"+assistant.ID)
			managedInstalled = append(managedInstalled, assistant.ID)
		} else {
			result.Skipped = append(result.Skipped, "assistant:"+assistant.ID)
		}
	}

	if err := addManagedAssistants(request.OutputDir, managedInstalled); err != nil {
		return usecase.InstallResult{}, err
	}

	return result, nil
}

func (g FilesystemGateway) ApplyAgentsPolicy(ctx context.Context, request usecase.InstallRequest, templateAgentsPath string) (usecase.InstallResult, error) {
	result := usecase.InstallResult{}
	if templateAgentsPath == "" {
		result.Skipped = append(result.Skipped, "agents:template-missing")
		return result, nil
	}

	outputDir := request.OutputDir
	if outputDir == "" {
		outputDir = "."
	}

	destination := filepath.Join(outputDir, "AGENTS.md")

	switch request.AgentsPolicy {
	case domain.AgentsPolicySkip:
		result.Skipped = append(result.Skipped, "agents:policy-skip")
		return result, nil
	case domain.AgentsPolicyIfMissing:
		if _, err := os.Stat(destination); err == nil {
			result.Skipped = append(result.Skipped, "agents:already-exists")
			return result, nil
		} else if !os.IsNotExist(err) {
			return usecase.InstallResult{}, fmt.Errorf("stat AGENTS destination: %w", err)
		}
	case domain.AgentsPolicyOverwrite:
		// continue
	default:
		return usecase.InstallResult{}, fmt.Errorf("invalid agents policy: %q", request.AgentsPolicy)
	}

	renderedContent, err := g.renderAgentsTemplate(ctx, request.OutputDir, templateAgentsPath)
	if err != nil {
		return usecase.InstallResult{}, err
	}

	output, err := writeFileWithOverwrite(destination, renderedContent)
	if err != nil {
		return usecase.InstallResult{}, err
	}

	if output.Created {
		result.Installed = append(result.Installed, "agents")
	} else {
		result.Skipped = append(result.Skipped, "agents")
	}

	return result, nil
}

func (g FilesystemGateway) renderAgentsTemplate(ctx context.Context, outputDir, templateAgentsPath string) ([]byte, error) {
	templateContent, err := os.ReadFile(templateAgentsPath)
	if err != nil {
		return nil, fmt.Errorf("read AGENTS template: %w", err)
	}

	projectContext := domain.ProjectContext{
		ProjectRoot: resolveProjectRoot(outputDir),
	}

	manifestPath := resolveProjectContextLayout(outputDir).ManifestPath
	if _, err := os.Stat(manifestPath); err == nil {
		loadedContext, loadErr := g.LoadProjectContext(ctx, outputDir)
		if loadErr != nil {
			return nil, loadErr
		}
		projectContext = loadedContext
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat project context manifest: %w", err)
	}

	rendered := applyAgentsTemplateContext(string(templateContent), projectContext)
	return []byte(rendered), nil
}

func applyAgentsTemplateContext(templateContent string, projectContext domain.ProjectContext) string {
	title := fallbackValue(projectContext.Title, "Projeto sem titulo informado")
	description := fallbackValue(projectContext.Description, "Contexto de negocio nao informado no heimdall start.")
	target := fallbackValue(string(projectContext.Target), "nao-definido")
	projectRoot := fallbackValue(projectContext.ProjectRoot, ".")
	documentsSummary := summarizeDocumentation(projectContext.Documentation)
	persona := buildSquadPersona(projectContext)

	replacements := map[string]string{
		"{{PROJECT_TITLE}}":        title,
		"{{PROJECT_DESCRIPTION}}":  description,
		"{{TARGET_PLATFORM}}":      target,
		"{{PROJECT_ROOT}}":         projectRoot,
		"{{PROJECT_DOCS_SUMMARY}}": documentsSummary,
		"{{SQUAD_PERSONA}}":        persona,
	}

	rendered := templateContent
	for marker, replacement := range replacements {
		rendered = strings.ReplaceAll(rendered, marker, replacement)
	}
	return rendered
}

func summarizeDocumentation(documents []string) string {
	if len(documents) == 0 {
		return "- Nenhuma documentacao registrada no heimdall start."
	}

	limit := len(documents)
	if limit > 3 {
		limit = 3
	}

	lines := make([]string, 0, limit+1)
	for i := 0; i < limit; i++ {
		lines = append(lines, "- "+strings.TrimSpace(documents[i]))
	}
	if len(documents) > limit {
		lines = append(lines, fmt.Sprintf("- ... e mais %d fonte(s) de contexto.", len(documents)-limit))
	}

	return strings.Join(lines, "\n")
}

func buildSquadPersona(projectContext domain.ProjectContext) string {
	text := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		projectContext.Title,
		projectContext.Description,
		strings.Join(projectContext.Documentation, " "),
	}, " ")))

	switch {
	case strings.Contains(text, "api"), strings.Contains(text, "backend"), strings.Contains(text, "microservice"), strings.Contains(text, "software"):
		return "Lideranca tecnica de engenharia orientada a produto, com foco em arquitetura evolutiva e entregas incrementais."
	case strings.Contains(text, "instagram"), strings.Contains(text, "linkedin"), strings.Contains(text, "conteudo"), strings.Contains(text, "marketing"):
		return "Lideranca de squad de conteudo orientada a estrategia, distribuicao e aprendizado continuo por metricas."
	case strings.Contains(text, "vendas"), strings.Contains(text, "comercial"), strings.Contains(text, "crm"), strings.Contains(text, "pipeline"):
		return "Lideranca de operacao comercial orientada a processo, clareza de funil e melhoria continua por dados."
	case strings.Contains(text, "produto"), strings.Contains(text, "discovery"), strings.Contains(text, "roadmap"), strings.Contains(text, "pesquisa"):
		return "Lideranca de produto orientada a descoberta, priorizacao baseada em impacto e validacao de hipoteses."
	default:
		return "Lideranca de squad multidisciplinar orientada a problema, com fronteiras claras de responsabilidade e colaboracao entre especialistas."
	}
}

func fallbackValue(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

type targetLayout struct {
	SkillsDir     string
	AssistantsDir string
}

type projectContextLayout struct {
	RootDir      string
	DocsDir      string
	ManifestPath string
}

type heimdallLayout struct {
	RootDir     string
	TemplateDir string
}

type projectContextManifest struct {
	Target      domain.TargetPlatform    `yaml:"target"`
	ProjectRoot string                   `yaml:"project_root,omitempty"`
	Title       string                   `yaml:"title"`
	Description string                   `yaml:"description"`
	Documents   []projectContextDocument `yaml:"documents"`
}

type projectContextDocument struct {
	Label      string `yaml:"label"`
	Source     string `yaml:"source"`
	StoredPath string `yaml:"stored_path"`
	Kind       string `yaml:"kind"`
}

type fileWriteOutput struct {
	Path    string
	Created bool
}

func resolveLayout(target domain.TargetPlatform, outputDir string) (targetLayout, error) {
	if outputDir == "" {
		outputDir = "."
	}

	switch target {
	case domain.TargetAntigravity:
		return targetLayout{
			SkillsDir:     filepath.Join(outputDir, ".agent", "skills"),
			AssistantsDir: filepath.Join(outputDir, ".agent", "workflows"),
		}, nil
	case domain.TargetCodex:
		return targetLayout{
			SkillsDir:     filepath.Join(outputDir, ".codex", "skills"),
			AssistantsDir: filepath.Join(outputDir, ".codex", "assistants"),
		}, nil
	case domain.TargetClaude:
		return targetLayout{
			SkillsDir:     filepath.Join(outputDir, ".claude", "skills"),
			AssistantsDir: filepath.Join(outputDir, ".claude", "assistants"),
		}, nil
	case domain.TargetCursor:
		return targetLayout{
			SkillsDir:     filepath.Join(outputDir, ".cursor", "skills"),
			AssistantsDir: filepath.Join(outputDir, ".cursor", "assistants"),
		}, nil
	default:
		return targetLayout{}, fmt.Errorf("unsupported target: %q", target)
	}
}

func resolveProjectContextLayout(outputDir string) projectContextLayout {
	if outputDir == "" {
		outputDir = "."
	}

	rootDir := filepath.Join(outputDir, ".heimdall", "context")
	return projectContextLayout{
		RootDir:      rootDir,
		DocsDir:      filepath.Join(rootDir, "docs"),
		ManifestPath: filepath.Join(rootDir, "project-context.yaml"),
	}
}

func resolveProjectRoot(outputDir string) string {
	baseDir := strings.TrimSpace(outputDir)
	if baseDir == "" {
		baseDir = "."
	}

	absolute, err := filepath.Abs(baseDir)
	if err != nil {
		return baseDir
	}
	return absolute
}

func resolveHeimdallLayout(outputDir string) heimdallLayout {
	if outputDir == "" {
		outputDir = "."
	}

	rootDir := filepath.Join(outputDir, ".heimdall")
	return heimdallLayout{
		RootDir:     rootDir,
		TemplateDir: filepath.Join(rootDir, "template"),
	}
}

func resolveTemplateSourceDir(agentsTemplatePath string) string {
	path := strings.TrimSpace(agentsTemplatePath)
	if path == "" {
		return ""
	}

	dir := filepath.Dir(path)
	if filepath.Base(dir) == ".agent" {
		return filepath.Dir(dir)
	}
	return dir
}

func createDirWithIdempotency(destinationDir string, force bool) (bool, error) {
	if !force {
		if _, err := os.Stat(destinationDir); err == nil {
			return false, nil
		} else if !os.IsNotExist(err) {
			return false, fmt.Errorf("stat destination dir %q: %w", destinationDir, err)
		}
	} else {
		_ = os.RemoveAll(destinationDir)
	}

	if err := os.MkdirAll(destinationDir, 0o755); err != nil {
		return false, fmt.Errorf("create destination dir %q: %w", destinationDir, err)
	}
	return true, nil
}

func copyFileWithIdempotency(source, destination string, force bool) (bool, error) {
	if !force {
		if _, err := os.Stat(destination); err == nil {
			return false, nil
		} else if !os.IsNotExist(err) {
			return false, fmt.Errorf("stat destination file %q: %w", destination, err)
		}
	} else {
		_ = os.Remove(destination)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return false, fmt.Errorf("create destination dir for %q: %w", destination, err)
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return false, fmt.Errorf("open source file %q: %w", source, err)
	}
	defer sourceFile.Close()

	stat, err := sourceFile.Stat()
	if err != nil {
		return false, fmt.Errorf("stat source file %q: %w", source, err)
	}

	destinationFile, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, stat.Mode().Perm())
	if err != nil {
		return false, fmt.Errorf("open destination file %q: %w", destination, err)
	}
	defer destinationFile.Close()

	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		return false, fmt.Errorf("copy file %q to %q: %w", source, destination, err)
	}

	return true, nil
}

func writeFileWithIdempotency(destination string, content []byte, force bool) (fileWriteOutput, error) {
	if !force {
		if _, err := os.Stat(destination); err == nil {
			return fileWriteOutput{Path: destination, Created: false}, nil
		} else if !os.IsNotExist(err) {
			return fileWriteOutput{}, fmt.Errorf("stat destination file %q: %w", destination, err)
		}
	} else {
		_ = os.Remove(destination)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fileWriteOutput{}, fmt.Errorf("create destination dir for %q: %w", destination, err)
	}

	if err := os.WriteFile(destination, content, 0o644); err != nil {
		return fileWriteOutput{}, fmt.Errorf("write file %q: %w", destination, err)
	}

	return fileWriteOutput{Path: destination, Created: true}, nil
}

func writeFileWithOverwrite(destination string, content []byte) (fileWriteOutput, error) {
	created := true
	if _, err := os.Stat(destination); err == nil {
		created = false
	} else if !os.IsNotExist(err) {
		return fileWriteOutput{}, fmt.Errorf("stat destination file %q: %w", destination, err)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fileWriteOutput{}, fmt.Errorf("create destination dir for %q: %w", destination, err)
	}

	if err := os.WriteFile(destination, content, 0o644); err != nil {
		return fileWriteOutput{}, fmt.Errorf("write file %q: %w", destination, err)
	}

	return fileWriteOutput{Path: destination, Created: created}, nil
}

func copyDirWithIdempotency(sourceDir, destinationDir string, force bool) (bool, error) {
	created, err := createDirWithIdempotency(destinationDir, force)
	if err != nil {
		return false, err
	}

	err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relativePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		destinationPath := filepath.Join(destinationDir, relativePath)
		if d.IsDir() {
			return os.MkdirAll(destinationPath, 0o755)
		}

		_, err = copyFileWithIdempotency(path, destinationPath, true)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("copy directory %q to %q: %w", sourceDir, destinationDir, err)
	}

	return created, nil
}

func appendFileOutcome(result *usecase.StartResult, output fileWriteOutput) {
	if output.Created {
		result.Created = append(result.Created, output.Path)
		return
	}

	result.Skipped = append(result.Skipped, output.Path)
}

func appendOverwrittenFileOutcome(result *usecase.StartResult, output fileWriteOutput) {
	if output.Created {
		result.Created = append(result.Created, output.Path)
		return
	}

	result.Updated = append(result.Updated, output.Path)
}

func appendInitFileOutcome(result *usecase.InitResult, output fileWriteOutput) {
	if output.Created {
		result.Created = append(result.Created, output.Path)
		return
	}

	result.Skipped = append(result.Skipped, output.Path)
}

type templateToolYAML struct {
	Type         string             `yaml:"type"`
	Category     string             `yaml:"category"`
	Categories   []string           `yaml:"categories"`
	ID           string             `yaml:"id"`
	Name         string             `yaml:"name"`
	Description  string             `yaml:"description"`
	Instructions string             `yaml:"instructions"`
	Skills       []string           `yaml:"skills"`
	Inputs       []domain.InputSpec `yaml:"inputs"`
	Tools        []string           `yaml:"tools"`
	Tags         []string           `yaml:"tags"`
	Metadata     map[string]string  `yaml:"metadata"`
}

func parseTemplateToolYAML(path string) (templateToolYAML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return templateToolYAML{}, fmt.Errorf("read tool %q: %w", path, err)
	}

	var tool templateToolYAML
	if err := yaml.Unmarshal(data, &tool); err != nil {
		return templateToolYAML{}, fmt.Errorf("parse tool YAML %q: %w", path, err)
	}

	tool.Type = strings.TrimSpace(tool.Type)
	tool.Category = strings.TrimSpace(tool.Category)
	if len(tool.Categories) == 0 && tool.Category != "" {
		tool.Categories = []string{tool.Category}
	}
	tool.Categories = dedupeAndTrim(tool.Categories)
	return tool, nil
}

func normalizeTemplateToolType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "assistant", "assitent":
		return "assistant"
	default:
		return normalized
	}
}

func hasCategory(categories []string, desired string) bool {
	want := strings.ToLower(strings.TrimSpace(desired))
	for _, category := range categories {
		if strings.ToLower(strings.TrimSpace(category)) == want {
			return true
		}
	}
	return false
}

func buildPlatformAssistantSkillInstructions(instructions string, associatedSkills []string) string {
	base := strings.TrimSpace(instructions)
	if len(associatedSkills) == 0 {
		return base
	}

	var builder strings.Builder
	builder.WriteString(base)
	builder.WriteString("\n\n## Associated Skills\n\n")
	for _, skill := range dedupeAndTrim(associatedSkills) {
		builder.WriteString("- ")
		builder.WriteString(skill)
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func dedupeAndTrim(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func persistDocumentationEntry(layout projectContextLayout, index int, entry string, force bool) (projectContextDocument, fileWriteOutput, error) {
	source := strings.TrimSpace(entry)
	baseName := fmt.Sprintf("%02d", index+1)

	if source != "" {
		if stat, err := os.Stat(source); err == nil && !stat.IsDir() {
			storedName := baseName + "-" + filepath.Base(source)
			storedPath := filepath.Join(layout.DocsDir, storedName)
			copied, copyErr := copyFileWithIdempotency(source, storedPath, force)
			if copyErr != nil {
				return projectContextDocument{}, fileWriteOutput{}, copyErr
			}

			return projectContextDocument{
				Label:      filepath.Base(source),
				Source:     source,
				StoredPath: filepath.ToSlash(filepath.Join("docs", storedName)),
				Kind:       "file",
			}, fileWriteOutput{Path: storedPath, Created: copied}, nil
		}
	}

	storedName := fmt.Sprintf("%s-note.md", baseName)
	storedPath := filepath.Join(layout.DocsDir, storedName)
	output, err := writeFileWithIdempotency(storedPath, []byte(source+"\n"), force)
	if err != nil {
		return projectContextDocument{}, fileWriteOutput{}, err
	}

	return projectContextDocument{
		Label:      fmt.Sprintf("note-%02d", index+1),
		Source:     source,
		StoredPath: filepath.ToSlash(filepath.Join("docs", storedName)),
		Kind:       "note",
	}, output, nil
}

func (g FilesystemGateway) copyTemplateSnapshot(outputDir string, force bool) (fileWriteOutput, error) {
	sourceDir := strings.TrimSpace(g.templateSourceDir)
	if sourceDir == "" {
		return fileWriteOutput{}, nil
	}

	heimdall := resolveHeimdallLayout(outputDir)
	created, err := copyDirWithIdempotency(sourceDir, heimdall.TemplateDir, force)
	if err != nil {
		return fileWriteOutput{}, fmt.Errorf("copy template snapshot to %q: %w", heimdall.TemplateDir, err)
	}

	return fileWriteOutput{Path: heimdall.TemplateDir, Created: created}, nil
}

func installAssistantWrapper(layout targetLayout, force bool, assistant usecase.AssistantAsset) (fileWriteOutput, error) {
	wrapperName := "assistant-" + assistant.ID
	wrapperPath := filepath.Join(layout.SkillsDir, wrapperName, "SKILL.md")
	content := []byte(buildAssistantSkillMarkdown(assistant))
	output, err := writeFileWithIdempotency(wrapperPath, content, force)
	if err != nil {
		return fileWriteOutput{}, fmt.Errorf("create assistant wrapper for %q: %w", assistant.ID, err)
	}
	return output, nil
}

func ensureSkillMarkdown(destinationDir string, skill usecase.SkillAsset, force bool) error {
	skillMarkdownPath := filepath.Join(destinationDir, "SKILL.md")

	if skill.Contract == nil {
		if _, err := os.Stat(skillMarkdownPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("skill %q must define skill.yaml/skill.yml or provide SKILL.md", skill.Name)
			}
			return fmt.Errorf("stat skill markdown %q: %w", skillMarkdownPath, err)
		}
		return nil
	}

	content := []byte(buildSkillMarkdownFromContract(skill))
	if _, err := writeFileWithIdempotency(skillMarkdownPath, content, force); err != nil {
		return fmt.Errorf("materialize SKILL.md for skill %q: %w", skill.Name, err)
	}
	return nil
}

func ensureGitignoreHasHeimdall(outputDir string) (bool, string, error) {
	if outputDir == "" {
		outputDir = "."
	}

	gitignorePath := filepath.Join(outputDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			if writeErr := os.WriteFile(gitignorePath, []byte(".heimdall\n"), 0o644); writeErr != nil {
				return false, "", fmt.Errorf("write .gitignore %q: %w", gitignorePath, writeErr)
			}
			return true, gitignorePath, nil
		}
		return false, "", fmt.Errorf("read .gitignore %q: %w", gitignorePath, err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == ".heimdall" {
			return false, gitignorePath, nil
		}
	}

	content := string(data)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += ".heimdall\n"

	if writeErr := os.WriteFile(gitignorePath, []byte(content), 0o644); writeErr != nil {
		return false, "", fmt.Errorf("update .gitignore %q: %w", gitignorePath, writeErr)
	}

	return true, gitignorePath, nil
}

func buildSkillMarkdownFromContract(skill usecase.SkillAsset) string {
	contract := skill.Contract
	name := strings.TrimSpace(skill.Name)
	description := ""
	instructions := ""
	if contract != nil {
		if strings.TrimSpace(contract.Name) != "" {
			name = strings.TrimSpace(contract.Name)
		}
		description = strings.TrimSpace(contract.Description)
		instructions = strings.TrimSpace(contract.Instructions)
	}

	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("name: ")
	builder.WriteString(name)
	builder.WriteString("\n")
	builder.WriteString("description: ")
	builder.WriteString(description)
	builder.WriteString("\n")
	builder.WriteString("---\n\n")
	builder.WriteString(instructions)
	builder.WriteString("\n")
	return builder.String()
}

func buildAssistantSkillMarkdown(assistant usecase.AssistantAsset) string {
	name := strings.TrimSpace(assistant.Name)
	if name == "" {
		name = assistant.ID
	}

	description := strings.TrimSpace(assistant.Description)
	if description == "" {
		description = "Assistant workflow wrapper."
	}

	instructions := strings.TrimSpace(assistant.Instructions)
	if instructions == "" {
		instructions = "Execute this assistant workflow using the associated skills and project context."
	}

	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("name: ")
	builder.WriteString(name)
	builder.WriteString("\n")
	builder.WriteString("description: ")
	builder.WriteString(description)
	builder.WriteString("\n")
	builder.WriteString("---\n\n")
	builder.WriteString("# Orchestrator Wrapper\n\n")
	builder.WriteString("Use this skill to execute orchestrator `")
	builder.WriteString(assistant.ID)
	builder.WriteString("`.\n\n")
	builder.WriteString("## Orchestration Instructions\n\n")
	builder.WriteString(instructions)
	builder.WriteString("\n\n")

	if len(assistant.Skills) > 0 {
		builder.WriteString("## Associated Skills\n\n")
		for _, skill := range assistant.Skills {
			builder.WriteString("- ")
			builder.WriteString(skill)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}
