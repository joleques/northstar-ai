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
		Title:         manifest.Title,
		Description:   manifest.Description,
		Documentation: documentation,
	}.Normalized()

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

	for _, dir := range []string{layout.SkillsDir, layout.AssistantsDir} {
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

	currentPlatformSkills, err := g.loadPlatformSkillsFromToolsDir(filepath.Join(sourceDir, "tools"))
	if err != nil {
		return usecase.UpdateAppResult{}, err
	}

	previousPlatformSkills, err := g.loadPlatformSkillsFromToolsDir(filepath.Join(resolveHeimdallLayout(request.OutputDir).TemplateDir, "tools"))
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

	templateOutput, templateErr := g.copyTemplateSnapshot(request.OutputDir, true)
	if templateErr != nil {
		return usecase.UpdateAppResult{}, templateErr
	}
	if templateOutput.Path != "" {
		result.Installed = append(result.Installed, templateOutput.Path)
	}

	skillsResult, err := g.InstallSkills(ctx, usecase.InstallRequest{
		Target:       request.Target,
		Force:        true,
		OutputDir:    request.OutputDir,
		SkipWrappers: true,
	}, currentPlatformSkills)
	if err != nil {
		return usecase.UpdateAppResult{}, err
	}

	result.Installed = append(result.Installed, skillsResult.Installed...)
	result.Skipped = append(result.Skipped, skillsResult.Skipped...)
	result.Failed = append(result.Failed, skillsResult.Failed...)
	result.Warnings = append(result.Warnings, skillsResult.Warnings...)

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
		Target:    request.Target,
		Documents: []projectContextDocument{},
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
	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []usecase.SkillAsset{}, nil
		}
		return nil, fmt.Errorf("read tools dir for platform auto-install: %w", err)
	}

	skills := make([]usecase.SkillAsset, 0)
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

			skills = append(skills, usecase.SkillAsset{
				Name: skillName,
				Contract: &usecase.SkillContract{
					Name:         skillName,
					Description:  strings.TrimSpace(tool.Description),
					Instructions: strings.TrimSpace(tool.Instructions),
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

			skills = append(skills, usecase.SkillAsset{
				Name: assistantID,
				Contract: &usecase.SkillContract{
					Name:         assistantName,
					Description:  strings.TrimSpace(tool.Description),
					Instructions: buildPlatformAssistantSkillInstructions(tool.Instructions, tool.Skills),
				},
			})
		default:
			return nil, fmt.Errorf("validate platform tool %q: unsupported type %q", toolPath, tool.Type)
		}
	}

	return skills, nil
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
		} else {
			result.Skipped = append(result.Skipped, "skill:"+skill.Name)
		}
	}

	return result, nil
}

func (g FilesystemGateway) InstallAssistants(_ context.Context, request usecase.InstallRequest, assistants []usecase.AssistantAsset) (usecase.InstallResult, error) {
	layout, err := resolveLayout(request.Target, request.OutputDir)
	if err != nil {
		return usecase.InstallResult{}, err
	}

	if err := os.MkdirAll(layout.AssistantsDir, 0o755); err != nil {
		return usecase.InstallResult{}, fmt.Errorf("create assistants dir: %w", err)
	}

	result := usecase.InstallResult{}
	for _, assistant := range assistants {
		ext := filepath.Ext(assistant.SourcePath)
		if ext == "" {
			ext = ".yaml"
		}
		dest := filepath.Join(layout.AssistantsDir, assistant.ID+ext)

		copied, err := copyFileWithIdempotency(assistant.SourcePath, dest, request.Force)
		if err != nil {
			result.Failed = append(result.Failed, "assistant:"+assistant.ID)
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}

		if copied {
			result.Installed = append(result.Installed, "assistant:"+assistant.ID)
		} else {
			result.Skipped = append(result.Skipped, "assistant:"+assistant.ID)
		}

		if request.Target == domain.TargetCodex && !request.SkipWrappers {
			wrapperOutput, wrapperErr := installCodexAssistantWrapper(layout, request.Force, assistant)
			if wrapperErr != nil {
				result.Failed = append(result.Failed, "assistant-wrapper:"+assistant.ID)
				result.Warnings = append(result.Warnings, wrapperErr.Error())
				continue
			}

			if wrapperOutput.Created {
				result.Installed = append(result.Installed, "assistant-wrapper:"+assistant.ID)
			} else {
				result.Skipped = append(result.Skipped, "assistant-wrapper:"+assistant.ID)
			}
		}
	}

	return result, nil
}

func (g FilesystemGateway) ApplyAgentsPolicy(_ context.Context, request usecase.InstallRequest, templateAgentsPath string) (usecase.InstallResult, error) {
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

	copied, err := copyFileWithIdempotency(templateAgentsPath, destination, true)
	if err != nil {
		return usecase.InstallResult{}, err
	}

	if copied {
		result.Installed = append(result.Installed, "agents")
	} else {
		result.Skipped = append(result.Skipped, "agents")
	}

	return result, nil
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

func installCodexAssistantWrapper(layout targetLayout, force bool, assistant usecase.AssistantAsset) (fileWriteOutput, error) {
	wrapperName := "assistant-" + assistant.ID
	wrapperPath := filepath.Join(layout.SkillsDir, wrapperName, "SKILL.md")
	content := []byte(buildCodexAssistantSkillMarkdown(assistant))
	output, err := writeFileWithIdempotency(wrapperPath, content, force)
	if err != nil {
		return fileWriteOutput{}, fmt.Errorf("create codex assistant wrapper for %q: %w", assistant.ID, err)
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

func buildCodexAssistantSkillMarkdown(assistant usecase.AssistantAsset) string {
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
	builder.WriteString("# Assistant Wrapper\n\n")
	builder.WriteString("Use this skill to execute assistant `")
	builder.WriteString(assistant.ID)
	builder.WriteString("`.\n\n")
	builder.WriteString("## Workflow Instructions\n\n")
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
