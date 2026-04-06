package usecase_test

import (
	"context"
	"testing"

	"github.com/heimdall-app/heimdall/src/domain"
	usecase "github.com/heimdall-app/heimdall/src/use_case"
)

type fakeUpdateAppGateway struct {
	called  bool
	request usecase.UpdateAppRequest
	result  usecase.UpdateAppResult
	err     error
}

func (f *fakeUpdateAppGateway) UpdateApp(_ context.Context, request usecase.UpdateAppRequest) (usecase.UpdateAppResult, error) {
	f.called = true
	f.request = request
	return f.result, f.err
}

type fakeProjectContextGateway struct {
	context domain.ProjectContext
	err     error
}

func (f *fakeProjectContextGateway) LoadProjectContext(_ context.Context, _ string) (domain.ProjectContext, error) {
	return f.context, f.err
}

func TestUpdateAppUseCaseExecuteWithExplicitTarget(t *testing.T) {
	t.Parallel()

	gateway := &fakeUpdateAppGateway{
		result: usecase.UpdateAppResult{Installed: []string{"skill:heimdall-install"}},
	}

	uc := usecase.NewUpdateAppUseCase(gateway, nil)
	result, err := uc.Execute(context.Background(), usecase.UpdateAppRequest{Target: domain.TargetCodex})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !gateway.called {
		t.Fatal("expected gateway to be called")
	}
	if gateway.request.Target != domain.TargetCodex {
		t.Fatalf("expected target codex, got %q", gateway.request.Target)
	}
	if len(result.Installed) != 1 {
		t.Fatalf("expected one installed item, got %d", len(result.Installed))
	}
}

func TestUpdateAppUseCaseExecuteLoadsTargetFromProjectContext(t *testing.T) {
	t.Parallel()

	gateway := &fakeUpdateAppGateway{
		result: usecase.UpdateAppResult{Removed: []string{"skill:heimdall-start"}},
	}
	projectContext := &fakeProjectContextGateway{
		context: domain.ProjectContext{Target: domain.TargetClaude},
	}

	uc := usecase.NewUpdateAppUseCase(gateway, projectContext)
	_, err := uc.Execute(context.Background(), usecase.UpdateAppRequest{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gateway.request.Target != domain.TargetClaude {
		t.Fatalf("expected target claude from project context, got %q", gateway.request.Target)
	}
}
