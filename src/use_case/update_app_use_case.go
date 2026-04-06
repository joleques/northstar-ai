package usecase

import (
	"context"
	"fmt"
)

type UpdateAppGateway interface {
	UpdateApp(ctx context.Context, request UpdateAppRequest) (UpdateAppResult, error)
}

type UpdateAppUseCase struct {
	gateway        UpdateAppGateway
	projectContext ProjectContextGateway
}

func NewUpdateAppUseCase(gateway UpdateAppGateway, projectContext ProjectContextGateway) UpdateAppUseCase {
	return UpdateAppUseCase{
		gateway:        gateway,
		projectContext: projectContext,
	}
}

func (uc UpdateAppUseCase) Execute(ctx context.Context, request UpdateAppRequest) (UpdateAppResult, error) {
	if request.Target == "" {
		if uc.projectContext == nil {
			return UpdateAppResult{}, fmt.Errorf("target is required when project context is unavailable")
		}

		projectContext, err := uc.projectContext.LoadProjectContext(ctx, request.OutputDir)
		if err != nil {
			return UpdateAppResult{}, err
		}

		request.Target = projectContext.Target
	}

	return uc.gateway.UpdateApp(ctx, request)
}

var _ UpdateApp = UpdateAppUseCase{}
