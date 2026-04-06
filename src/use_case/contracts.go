package usecase

import (
	"context"

	"github.com/joleques/northstar-ai/src/domain"
)

type InstallRequest struct {
	Target       domain.TargetPlatform
	Assistants   []string
	Category     string
	AgentsPolicy domain.AgentsPolicy
	Force        bool
	OutputDir    string
	SkipWrappers bool
}

type InstallResult struct {
	Installed []string
	Skipped   []string
	Failed    []string
	Warnings  []string
}

type InstallAssistant interface {
	Execute(ctx context.Context, request InstallRequest) (InstallResult, error)
}

type InitRequest struct {
	Target       domain.TargetPlatform
	AgentsPolicy domain.AgentsPolicy
	Force        bool
	OutputDir    string
}

type InitResult struct {
	Created  []string
	Skipped  []string
	Warnings []string
}

type InitTarget interface {
	Execute(ctx context.Context, request InitRequest) (InitResult, error)
}

type StartRequest struct {
	Target        domain.TargetPlatform
	Title         string
	Description   string
	Documentation []string
	OutputDir     string
	Force         bool
}

type StartResult struct {
	Created  []string
	Updated  []string
	Skipped  []string
	Warnings []string
}

type StartProject interface {
	Execute(ctx context.Context, request StartRequest) (StartResult, error)
}

type ListLibraryRequest struct {
	IncludeSkills bool
	Category      string
	OutputDir     string
}

type AssistantLibraryItem struct {
	ID          string
	Name        string
	Description string
	Skills      []string
	Categories  []string
}

type SkillLibraryItem struct {
	ID          string
	Description string
	Categories  []string
}

type ListLibraryResult struct {
	Assistants []AssistantLibraryItem
	Skills     []SkillLibraryItem
}

type ListLibrary interface {
	Execute(ctx context.Context, request ListLibraryRequest) (ListLibraryResult, error)
}

type UpdateAppRequest struct {
	Target    domain.TargetPlatform
	OutputDir string
}

type UpdateAppResult struct {
	Removed   []string
	Installed []string
	Skipped   []string
	Failed    []string
	Warnings  []string
}

type UpdateApp interface {
	Execute(ctx context.Context, request UpdateAppRequest) (UpdateAppResult, error)
}
