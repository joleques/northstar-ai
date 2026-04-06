package usecase_test

import (
	"context"
	"testing"

	"github.com/joleques/northstar-ai/src/domain"
	usecase "github.com/joleques/northstar-ai/src/use_case"
)

type fakeInitGateway struct {
	called  bool
	request usecase.InitRequest
	result  usecase.InitResult
	err     error
}

func (f *fakeInitGateway) InitTarget(_ context.Context, request usecase.InitRequest) (usecase.InitResult, error) {
	f.called = true
	f.request = request
	return f.result, f.err
}

func TestInitTargetUseCaseExecute(t *testing.T) {
	t.Parallel()

	gateway := &fakeInitGateway{result: usecase.InitResult{Created: []string{".codex/skills"}}}
	uc := usecase.NewInitTargetUseCase(gateway)

	result, err := uc.Execute(context.Background(), usecase.InitRequest{Target: domain.TargetCodex})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !gateway.called {
		t.Fatal("expected gateway to be called")
	}
	if len(result.Created) != 1 {
		t.Fatalf("expected 1 created entry, got %d", len(result.Created))
	}
}
