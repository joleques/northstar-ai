package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/joleques/northstar-ai/src/domain"
	usecase "github.com/joleques/northstar-ai/src/use_case"
)

type fakeStartGateway struct {
	called  bool
	request usecase.StartRequest
	result  usecase.StartResult
	err     error
	context domain.ProjectContext
}

func (f *fakeStartGateway) SaveProjectContext(_ context.Context, request usecase.StartRequest) (usecase.StartResult, error) {
	f.called = true
	f.request = request
	return f.result, f.err
}

func (f *fakeStartGateway) LoadProjectContext(_ context.Context, _ string) (domain.ProjectContext, error) {
	if f.context.Target == "" {
		return domain.ProjectContext{}, fmt.Errorf("project context not found")
	}
	return f.context, nil
}

func TestStartProjectUseCaseExecute(t *testing.T) {
	t.Parallel()

	gateway := &fakeStartGateway{result: usecase.StartResult{Created: []string{".heimdall/context/project-context.yaml"}}}
	uc := usecase.NewStartProjectUseCase(gateway)

	result, err := uc.Execute(context.Background(), usecase.StartRequest{
		Target:        "  codex  ",
		Title:         "  Heimdall  ",
		Description:   "  Contexto base do projeto.  ",
		Documentation: []string{" README.md ", "README.md"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !gateway.called {
		t.Fatal("expected gateway to be called")
	}

	if gateway.request.Title != "Heimdall" {
		t.Fatalf("expected normalized title, got %q", gateway.request.Title)
	}

	if gateway.request.Target != "codex" {
		t.Fatalf("expected normalized target, got %q", gateway.request.Target)
	}

	if len(gateway.request.Documentation) != 1 {
		t.Fatalf("expected deduplicated documentation, got %d entries", len(gateway.request.Documentation))
	}

	if len(result.Created) != 1 {
		t.Fatalf("expected 1 created entry, got %d", len(result.Created))
	}
}

func TestStartProjectUseCaseRejectsInvalidContext(t *testing.T) {
	t.Parallel()

	gateway := &fakeStartGateway{}
	uc := usecase.NewStartProjectUseCase(gateway)

	_, err := uc.Execute(context.Background(), usecase.StartRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	if gateway.called {
		t.Fatal("gateway should not be called on invalid request")
	}
}

func TestStartProjectUseCaseLoadsTargetFromProjectContext(t *testing.T) {
	t.Parallel()

	gateway := &fakeStartGateway{
		result: usecase.StartResult{Created: []string{".heimdall/context/project-context.yaml"}},
		context: domain.ProjectContext{
			Target: domain.TargetCursor,
		},
	}
	uc := usecase.NewStartProjectUseCase(gateway)

	_, err := uc.Execute(context.Background(), usecase.StartRequest{
		Title:       "Heimdall",
		Description: "Contexto base do projeto.",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gateway.request.Target != domain.TargetCursor {
		t.Fatalf("expected target loaded from project context, got %q", gateway.request.Target)
	}
}
