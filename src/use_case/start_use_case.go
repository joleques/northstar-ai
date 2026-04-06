package usecase

import (
	"context"

	"github.com/joleques/northstar-ai/src/domain"
)

type StartGateway interface {
	SaveProjectContext(ctx context.Context, request StartRequest) (StartResult, error)
	LoadProjectContext(ctx context.Context, outputDir string) (domain.ProjectContext, error)
}

type StartProjectUseCase struct {
	gateway StartGateway
}

func NewStartProjectUseCase(gateway StartGateway) StartProjectUseCase {
	return StartProjectUseCase{gateway: gateway}
}

func (uc StartProjectUseCase) Execute(ctx context.Context, request StartRequest) (StartResult, error) {
	if request.Target == "" {
		existingContext, err := uc.gateway.LoadProjectContext(ctx, request.OutputDir)
		if err != nil {
			return StartResult{}, err
		}
		request.Target = existingContext.Target
	}

	context := domain.ProjectContext{
		Target:        request.Target,
		Title:         request.Title,
		Description:   request.Description,
		Documentation: request.Documentation,
	}.Normalized()

	if err := context.Validate(); err != nil {
		return StartResult{}, err
	}

	request.Target = context.Target
	request.Title = context.Title
	request.Description = context.Description
	request.Documentation = context.Documentation

	return uc.gateway.SaveProjectContext(ctx, request)
}

var _ StartProject = StartProjectUseCase{}
