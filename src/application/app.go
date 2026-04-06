package application

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	infrainstall "github.com/heimdall-app/heimdall/src/infra/install"
	infratemplate "github.com/heimdall-app/heimdall/src/infra/template"
	usecase "github.com/heimdall-app/heimdall/src/use_case"
)

type App struct {
	in                 io.Reader
	out                io.Writer
	installAssistantUC usecase.InstallAssistant
	initTargetUC       usecase.InitTarget
	startProjectUC     usecase.StartProject
	listLibraryUC      usecase.ListLibrary
	updateAppUC        usecase.UpdateApp
}

func NewApp(out io.Writer) App {
	templateRoot, err := infratemplate.ResolveTemplateRoot()
	if err != nil {
		templateRoot = ""
	}

	catalogGateway := infratemplate.NewCatalogGateway(templateRoot)
	installGateway := infrainstall.NewFilesystemGateway(filepath.Join(templateRoot, "AGENTS.md"))

	assistantUC := usecase.NewInstallAssistantUseCase(catalogGateway, installGateway, installGateway)
	initUC := usecase.NewInitTargetUseCase(installGateway)
	startUC := usecase.NewStartProjectUseCase(installGateway)
	listUC := usecase.NewListLibraryUseCase(catalogGateway)
	updateUC := usecase.NewUpdateAppUseCase(installGateway, installGateway)

	return App{
		in:                 os.Stdin,
		out:                out,
		installAssistantUC: assistantUC,
		initTargetUC:       initUC,
		startProjectUC:     startUC,
		listLibraryUC:      listUC,
		updateAppUC:        updateUC,
	}
}

func NewAppWithUseCases(in io.Reader, out io.Writer, installAssistantUC usecase.InstallAssistant, initTargetUC usecase.InitTarget, startProjectUC usecase.StartProject, listLibraryUC usecase.ListLibrary, updateAppUC usecase.UpdateApp) App {
	return App{
		in:                 in,
		out:                out,
		installAssistantUC: installAssistantUC,
		initTargetUC:       initTargetUC,
		startProjectUC:     startProjectUC,
		listLibraryUC:      listLibraryUC,
		updateAppUC:        updateAppUC,
	}
}

func (a App) Run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.out, "usage: heimdall <command> [args]")
		fmt.Fprintln(a.out, "commands: init, start, list-lib, install, update-app")
		return 1
	}

	switch args[0] {
	case "init":
		return a.runInit(args)
	case "start":
		return a.runStart(args)
	case "list-lib":
		return a.runListLibrary(args)
	case "install":
		return a.runInstall(args)
	case "update-app":
		return a.runUpdateApp(args)
	default:
		fmt.Fprintf(a.out, "error: unsupported command %q\n", args[0])
		fmt.Fprintln(a.out, "commands: init, start, list-lib, install, update-app")
		return 1
	}
}

func (a App) runInit(args []string) int {
	parsed, err := ParseInitArgs(args)
	if err != nil {
		fmt.Fprintf(a.out, "error: %v\n", err)
		fmt.Fprintln(a.out, "usage: heimdall init <codex|antigravity|claude|cursor> [--agents-policy <skip|if-missing|overwrite>] [--force] [--output <dir>]")
		return 1
	}

	result, err := a.initTargetUC.Execute(context.Background(), parsed.Request)
	if err != nil {
		fmt.Fprintf(a.out, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(
		a.out,
		"init (%s) completed: created=%d skipped=%d warnings=%d\n",
		parsed.Request.Target,
		len(result.Created),
		len(result.Skipped),
		len(result.Warnings),
	)

	return 0
}

func (a App) runStart(args []string) int {
	parsed, err := ParseStartArgs(args, nil)
	if err != nil {
		fmt.Fprintf(a.out, "error: %v\n", err)
		fmt.Fprintln(a.out, "usage: heimdall start [--target <codex|antigravity|claude|cursor>] [--title <value>] [--description <value>] [--doc <path-or-text>]... [--interactive] [--force] [--output <dir>]")
		return 1
	}

	if parsed.Interactive || needsInteractiveStartInput(parsed.Request) {
		if strings.TrimSpace(parsed.Request.Title) == "" {
			fmt.Fprintln(a.out, "Project title:")
		}
		if strings.TrimSpace(parsed.Request.Description) == "" {
			fmt.Fprintln(a.out, "Project description/context:")
		}

		parsed.Request, err = collectStartInput(a.in, parsed.Request)
		if err != nil {
			fmt.Fprintf(a.out, "error: %v\n", err)
			return 1
		}
	}

	result, err := a.startProjectUC.Execute(context.Background(), parsed.Request)
	if err != nil {
		fmt.Fprintf(a.out, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(
		a.out,
		"start completed: created=%d updated=%d skipped=%d warnings=%d\n",
		len(result.Created),
		len(result.Updated),
		len(result.Skipped),
		len(result.Warnings),
	)

	return 0
}

func (a App) runListLibrary(args []string) int {
	parsed, err := ParseListLibraryArgs(args)
	if err != nil {
		fmt.Fprintf(a.out, "error: %v\n", err)
		fmt.Fprintln(a.out, "usage: heimdall list-lib [--skills] [--category <software-architecture|media|documentation|platform>]")
		return 1
	}

	result, err := a.listLibraryUC.Execute(context.Background(), parsed.Request)
	if err != nil {
		fmt.Fprintf(a.out, "error: %v\n", err)
		return 1
	}

	includeSkills := parsed.Request.IncludeSkills || strings.TrimSpace(parsed.Request.Category) != ""
	if includeSkills {
		fmt.Fprintf(a.out, "list-lib completed: assistants=%d skills=%d\n", len(result.Assistants), len(result.Skills))
	} else {
		fmt.Fprintf(a.out, "list-lib completed: assistants=%d\n", len(result.Assistants))
	}

	for _, assistant := range result.Assistants {
		fmt.Fprintf(a.out, "- %s | %s | %s | skills=%s | categories=%s\n", assistant.ID, assistant.Name, assistant.Description, strings.Join(assistant.Skills, ","), strings.Join(assistant.Categories, ","))
	}
	if includeSkills {
		for _, skill := range result.Skills {
			fmt.Fprintf(a.out, "- skill:%s | %s | categories=%s\n", skill.ID, skill.Description, strings.Join(skill.Categories, ","))
		}
	}

	return 0
}

func (a App) runInstall(args []string) int {
	parsed, err := ParseCLIArgs(args)
	if err != nil {
		fmt.Fprintf(a.out, "error: %v\n", err)
		fmt.Fprintln(a.out, "usage: heimdall install [assistant-id ...] [--category <software-architecture|media|documentation|platform>] [--agents-policy <skip|if-missing|overwrite>] [--force] [--output <dir>]")
		fmt.Fprintln(a.out, "legacy: heimdall install <codex|antigravity|claude|cursor> [assistant-id ...] [--category <...>] [options]")
		return 1
	}

	result, err := a.installAssistantUC.Execute(context.Background(), parsed.Request)
	if err != nil {
		fmt.Fprintf(a.out, "error: %v\n", err)
		return 1
	}

	if parsed.Request.Target == "" {
		fmt.Fprintf(
			a.out,
			"install assistant completed: assistants=%d installed=%d skipped=%d failed=%d warnings=%d\n",
			len(parsed.Request.Assistants),
			len(result.Installed),
			len(result.Skipped),
			len(result.Failed),
			len(result.Warnings),
		)
	} else {
		fmt.Fprintf(
			a.out,
			"install assistant (%s) completed: assistants=%d installed=%d skipped=%d failed=%d warnings=%d\n",
			parsed.Request.Target,
			len(parsed.Request.Assistants),
			len(result.Installed),
			len(result.Skipped),
			len(result.Failed),
			len(result.Warnings),
		)
	}

	printInstallDetails(a.out, "installed", result.Installed)
	printInstallDetails(a.out, "skipped", result.Skipped)
	printInstallDetails(a.out, "failed", result.Failed)
	printInstallDetails(a.out, "warning", result.Warnings)

	if len(result.Failed) > 0 {
		return 1
	}

	return 0
}

func printInstallDetails(out io.Writer, label string, items []string) {
	for _, item := range items {
		fmt.Fprintf(out, "%s: %s\n", label, item)
	}
}

func (a App) runUpdateApp(args []string) int {
	parsed, err := ParseUpdateAppArgs(args)
	if err != nil {
		fmt.Fprintf(a.out, "error: %v\n", err)
		fmt.Fprintln(a.out, "usage: heimdall update-app [codex|antigravity|claude|cursor] [--output <dir>]")
		return 1
	}

	result, err := a.updateAppUC.Execute(context.Background(), parsed.Request)
	if err != nil {
		fmt.Fprintf(a.out, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(
		a.out,
		"update-app completed: removed=%d installed=%d skipped=%d failed=%d warnings=%d\n",
		len(result.Removed),
		len(result.Installed),
		len(result.Skipped),
		len(result.Failed),
		len(result.Warnings),
	)

	printInstallDetails(a.out, "removed", result.Removed)
	printInstallDetails(a.out, "installed", result.Installed)
	printInstallDetails(a.out, "skipped", result.Skipped)
	printInstallDetails(a.out, "failed", result.Failed)
	printInstallDetails(a.out, "warning", result.Warnings)

	if len(result.Failed) > 0 {
		return 1
	}

	return 0
}
