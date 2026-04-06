package application_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/joleques/northstar-ai/src/application"
	usecase "github.com/joleques/northstar-ai/src/use_case"
)

type mockInstallAssistant struct {
	called  bool
	request usecase.InstallRequest
	result  usecase.InstallResult
	err     error
}

func (m *mockInstallAssistant) Execute(_ context.Context, request usecase.InstallRequest) (usecase.InstallResult, error) {
	m.called = true
	m.request = request
	return m.result, m.err
}

type mockInitTarget struct {
	called  bool
	request usecase.InitRequest
	result  usecase.InitResult
	err     error
}

func (m *mockInitTarget) Execute(_ context.Context, request usecase.InitRequest) (usecase.InitResult, error) {
	m.called = true
	m.request = request
	return m.result, m.err
}

type mockStartProject struct {
	called  bool
	request usecase.StartRequest
	result  usecase.StartResult
	err     error
}

func (m *mockStartProject) Execute(_ context.Context, request usecase.StartRequest) (usecase.StartResult, error) {
	m.called = true
	m.request = request
	return m.result, m.err
}

type mockListLibrary struct {
	called  bool
	request usecase.ListLibraryRequest
	result  usecase.ListLibraryResult
	err     error
}

func (m *mockListLibrary) Execute(_ context.Context, request usecase.ListLibraryRequest) (usecase.ListLibraryResult, error) {
	m.called = true
	m.request = request
	return m.result, m.err
}

type mockUpdateApp struct {
	called  bool
	request usecase.UpdateAppRequest
	result  usecase.UpdateAppResult
	err     error
}

func (m *mockUpdateApp) Execute(_ context.Context, request usecase.UpdateAppRequest) (usecase.UpdateAppResult, error) {
	m.called = true
	m.request = request
	return m.result, m.err
}

func TestAppRunInstallAssistant(t *testing.T) {
	t.Parallel()

	installUC := &mockInstallAssistant{result: usecase.InstallResult{
		Installed: []string{"skill:researcher", "assistant:instagram-post-studio"},
		Skipped:   []string{"agents:already-exists"},
		Warnings:  []string{"assistant \"instagram-post-studio\" references missing skill \"designer\""},
	}}
	initUC := &mockInitTarget{}
	startUC := &mockStartProject{}
	listUC := &mockListLibrary{}
	updateUC := &mockUpdateApp{}
	var out bytes.Buffer
	app := application.NewAppWithUseCases(bytes.NewBuffer(nil), &out, installUC, initUC, startUC, listUC, updateUC)

	code := app.Run([]string{"install", "assistant", "codex", "instagram-post-studio"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !installUC.called {
		t.Fatal("expected install assistant use case to be called")
	}
	if initUC.called {
		t.Fatal("init use case should not be called on install command")
	}
	if startUC.called {
		t.Fatal("start use case should not be called on install command")
	}
	if listUC.called || updateUC.called {
		t.Fatal("list use case should not be called on install command")
	}

	output := out.String()
	for _, fragment := range []string{
		"install assistant (codex) completed",
		"installed: skill:researcher",
		"installed: assistant:instagram-post-studio",
		"skipped: agents:already-exists",
		"warning: assistant \"instagram-post-studio\" references missing skill \"designer\"",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got %s", fragment, output)
		}
	}
}

func TestAppRunInitTarget(t *testing.T) {
	t.Parallel()

	installUC := &mockInstallAssistant{}
	initUC := &mockInitTarget{}
	startUC := &mockStartProject{}
	listUC := &mockListLibrary{}
	updateUC := &mockUpdateApp{}
	var out bytes.Buffer
	app := application.NewAppWithUseCases(bytes.NewBuffer(nil), &out, installUC, initUC, startUC, listUC, updateUC)

	code := app.Run([]string{"init", "codex"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !initUC.called {
		t.Fatal("expected init use case to be called")
	}
	if installUC.called {
		t.Fatal("install use case should not be called on init command")
	}
	if startUC.called {
		t.Fatal("start use case should not be called on init command")
	}
	if listUC.called || updateUC.called {
		t.Fatal("list use case should not be called on init command")
	}
}

func TestAppRunStartProject(t *testing.T) {
	t.Parallel()

	installUC := &mockInstallAssistant{}
	initUC := &mockInitTarget{}
	startUC := &mockStartProject{}
	listUC := &mockListLibrary{}
	updateUC := &mockUpdateApp{}
	var out bytes.Buffer
	in := bytes.NewBufferString("Heimdall\nProject context\n")
	app := application.NewAppWithUseCases(in, &out, installUC, initUC, startUC, listUC, updateUC)

	code := app.Run([]string{"start"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !startUC.called {
		t.Fatal("expected start use case to be called")
	}
	if startUC.request.Target != "" {
		t.Fatalf("expected empty start target from parser, got %q", startUC.request.Target)
	}
	if installUC.called || initUC.called || listUC.called || updateUC.called {
		t.Fatal("install/init/list use cases should not be called on start command")
	}
}

func TestAppRunInstallAssistantReturnsErrorCodeWhenInstallHasFailures(t *testing.T) {
	t.Parallel()

	installUC := &mockInstallAssistant{result: usecase.InstallResult{
		Failed: []string{"assistant:broken-assistant"},
	}}
	initUC := &mockInitTarget{}
	startUC := &mockStartProject{}
	listUC := &mockListLibrary{}
	updateUC := &mockUpdateApp{}
	var out bytes.Buffer
	app := application.NewAppWithUseCases(bytes.NewBuffer(nil), &out, installUC, initUC, startUC, listUC, updateUC)

	code := app.Run([]string{"install", "assistant", "codex", "broken-assistant"})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	if !strings.Contains(out.String(), "failed: assistant:broken-assistant") {
		t.Fatalf("expected failed item in output, got %s", out.String())
	}
}

func TestAppRunListLibrary(t *testing.T) {
	t.Parallel()

	installUC := &mockInstallAssistant{}
	initUC := &mockInitTarget{}
	startUC := &mockStartProject{}
	listUC := &mockListLibrary{result: usecase.ListLibraryResult{
		Assistants: []usecase.AssistantLibraryItem{
			{ID: "assistant-a", Name: "Assistant A", Description: "desc", Skills: []string{"skill-a"}, Categories: []string{"platform"}},
		},
	}}
	updateUC := &mockUpdateApp{}
	var out bytes.Buffer
	app := application.NewAppWithUseCases(bytes.NewBuffer(nil), &out, installUC, initUC, startUC, listUC, updateUC)

	code := app.Run([]string{"list-lib"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !listUC.called {
		t.Fatal("expected list library use case to be called")
	}
	if installUC.called || initUC.called || startUC.called || updateUC.called {
		t.Fatal("other use cases should not be called on list-lib command")
	}
}

func TestAppRunListLibraryWithSkills(t *testing.T) {
	t.Parallel()

	installUC := &mockInstallAssistant{}
	initUC := &mockInitTarget{}
	startUC := &mockStartProject{}
	listUC := &mockListLibrary{result: usecase.ListLibraryResult{
		Assistants: []usecase.AssistantLibraryItem{
			{ID: "assistant-a", Name: "Assistant A", Description: "desc", Skills: []string{"skill-a"}, Categories: []string{"documentation"}},
		},
		Skills: []usecase.SkillLibraryItem{
			{ID: "skill-a", Description: "Skill A", Categories: []string{"documentation"}},
		},
	}}
	updateUC := &mockUpdateApp{}
	var out bytes.Buffer
	app := application.NewAppWithUseCases(bytes.NewBuffer(nil), &out, installUC, initUC, startUC, listUC, updateUC)

	code := app.Run([]string{"list-lib", "--skills", "--category", "documentation", "--output", "/tmp/client-project"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !listUC.called {
		t.Fatal("expected list library use case to be called")
	}
	if !listUC.request.IncludeSkills {
		t.Fatal("expected include skills request to be true")
	}
	if listUC.request.Category != "documentation" {
		t.Fatalf("expected category documentation, got %q", listUC.request.Category)
	}
	if listUC.request.OutputDir != "/tmp/client-project" {
		t.Fatalf("expected output dir /tmp/client-project, got %q", listUC.request.OutputDir)
	}

	output := out.String()
	for _, fragment := range []string{
		"list-lib completed: assistants=1 skills=1",
		"- assistant-a | Assistant A | desc | skills=skill-a | categories=documentation",
		"- skill:skill-a | Skill A | categories=documentation",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got %s", fragment, output)
		}
	}
}

func TestAppRunListLibraryWithCategoryOnlyIncludesSkills(t *testing.T) {
	t.Parallel()

	installUC := &mockInstallAssistant{}
	initUC := &mockInitTarget{}
	startUC := &mockStartProject{}
	listUC := &mockListLibrary{result: usecase.ListLibraryResult{
		Assistants: []usecase.AssistantLibraryItem{
			{ID: "assistant-a", Name: "Assistant A", Description: "desc", Skills: []string{"skill-a"}, Categories: []string{"documentation"}},
		},
		Skills: []usecase.SkillLibraryItem{
			{ID: "skill-a", Description: "Skill A", Categories: []string{"documentation"}},
		},
	}}
	updateUC := &mockUpdateApp{}
	var out bytes.Buffer
	app := application.NewAppWithUseCases(bytes.NewBuffer(nil), &out, installUC, initUC, startUC, listUC, updateUC)

	code := app.Run([]string{"list-lib", "--category", "documentation"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !listUC.called {
		t.Fatal("expected list library use case to be called")
	}
	if listUC.request.IncludeSkills {
		t.Fatal("expected include skills request to be false when only category is provided")
	}
	if listUC.request.Category != "documentation" {
		t.Fatalf("expected category documentation, got %q", listUC.request.Category)
	}

	output := out.String()
	for _, fragment := range []string{
		"list-lib completed: assistants=1 skills=1",
		"- assistant-a | Assistant A | desc | skills=skill-a | categories=documentation",
		"- skill:skill-a | Skill A | categories=documentation",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got %s", fragment, output)
		}
	}
}

func TestAppRunInvalidArgs(t *testing.T) {
	t.Parallel()

	installUC := &mockInstallAssistant{}
	initUC := &mockInitTarget{}
	startUC := &mockStartProject{}
	listUC := &mockListLibrary{}
	updateUC := &mockUpdateApp{}
	var out bytes.Buffer
	app := application.NewAppWithUseCases(bytes.NewBuffer(nil), &out, installUC, initUC, startUC, listUC, updateUC)

	code := app.Run([]string{"banana"})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	if installUC.called || initUC.called || startUC.called || listUC.called || updateUC.called {
		t.Fatal("expected no use case calls on invalid command")
	}
}

func TestAppRunUpdateApp(t *testing.T) {
	t.Parallel()

	installUC := &mockInstallAssistant{}
	initUC := &mockInitTarget{}
	startUC := &mockStartProject{}
	listUC := &mockListLibrary{}
	updateUC := &mockUpdateApp{result: usecase.UpdateAppResult{
		Removed:   []string{"skill:heimdall-install"},
		Installed: []string{"skill:heimdall-install"},
	}}

	var out bytes.Buffer
	app := application.NewAppWithUseCases(bytes.NewBuffer(nil), &out, installUC, initUC, startUC, listUC, updateUC)

	code := app.Run([]string{"update-app", "codex"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !updateUC.called {
		t.Fatal("expected update app use case to be called")
	}
	if updateUC.request.Target != "codex" {
		t.Fatalf("expected target codex, got %q", updateUC.request.Target)
	}
	if installUC.called || initUC.called || startUC.called || listUC.called {
		t.Fatal("other use cases should not be called on update-app command")
	}

	output := out.String()
	for _, fragment := range []string{
		"update-app completed",
		"removed: skill:heimdall-install",
		"installed: skill:heimdall-install",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got %s", fragment, output)
		}
	}
}

func TestAppRunUpdateAppReturnsErrorCodeWhenHasFailures(t *testing.T) {
	t.Parallel()

	installUC := &mockInstallAssistant{}
	initUC := &mockInitTarget{}
	startUC := &mockStartProject{}
	listUC := &mockListLibrary{}
	updateUC := &mockUpdateApp{result: usecase.UpdateAppResult{
		Failed: []string{"skill:heimdall-install"},
	}}

	var out bytes.Buffer
	app := application.NewAppWithUseCases(bytes.NewBuffer(nil), &out, installUC, initUC, startUC, listUC, updateUC)

	code := app.Run([]string{"update-app"})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(out.String(), "failed: skill:heimdall-install") {
		t.Fatalf("expected failed item in output, got %s", out.String())
	}
}
