package usecase_test

import (
	"context"
	"testing"

	usecase "github.com/joleques/northstar-ai/src/use_case"
)

func TestListLibraryUseCaseExecute(t *testing.T) {
	t.Parallel()

	uc := usecase.NewListLibraryUseCase(fakeCatalogGateway{
		catalog: usecase.Catalog{
			Skills: []usecase.SkillAsset{
				{Name: "skill-z", Contract: &usecase.SkillContract{Description: "Ultima skill"}, Categories: []string{"media"}},
				{Name: "skill-a", Contract: &usecase.SkillContract{Description: "Primeira skill"}, Categories: []string{"documentation"}},
			},
			Assistants: []usecase.AssistantAsset{
				{ID: "zeta", Name: "Zeta", Description: "Ultimo", Skills: []string{"skill-z"}, Categories: []string{"media"}},
				{ID: "alpha", Name: "Alpha", Description: "Primeiro", Skills: []string{"skill-a", "skill-b"}, Categories: []string{"documentation"}},
			},
		},
	})

	result, err := uc.Execute(context.Background(), usecase.ListLibraryRequest{IncludeSkills: true, Category: "documentation"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Assistants) != 1 {
		t.Fatalf("expected 1 assistant after category filter, got %d", len(result.Assistants))
	}

	if result.Assistants[0].ID != "alpha" {
		t.Fatalf("expected assistants sorted by id, got first %q", result.Assistants[0].ID)
	}

	if result.Assistants[0].Name != "Alpha" {
		t.Fatalf("expected assistant name to be preserved, got %q", result.Assistants[0].Name)
	}

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill after category filter, got %d", len(result.Skills))
	}

	if result.Skills[0].ID != "skill-a" {
		t.Fatalf("expected skills sorted by id, got first %q", result.Skills[0].ID)
	}
}

func TestListLibraryUseCaseExecuteCategoryIncludesSkillsWithoutFlag(t *testing.T) {
	t.Parallel()

	uc := usecase.NewListLibraryUseCase(fakeCatalogGateway{
		catalog: usecase.Catalog{
			Skills: []usecase.SkillAsset{
				{Name: "skill-z", Contract: &usecase.SkillContract{Description: "Ultima skill"}, Categories: []string{"media"}},
				{Name: "skill-a", Contract: &usecase.SkillContract{Description: "Primeira skill"}, Categories: []string{"documentation"}},
			},
			Assistants: []usecase.AssistantAsset{
				{ID: "zeta", Name: "Zeta", Description: "Ultimo", Skills: []string{"skill-z"}, Categories: []string{"media"}},
				{ID: "alpha", Name: "Alpha", Description: "Primeiro", Skills: []string{"skill-a"}, Categories: []string{"documentation"}},
			},
		},
	})

	result, err := uc.Execute(context.Background(), usecase.ListLibraryRequest{Category: "documentation"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Assistants) != 1 || result.Assistants[0].ID != "alpha" {
		t.Fatalf("expected filtered assistant alpha, got %#v", result.Assistants)
	}

	if len(result.Skills) != 1 || result.Skills[0].ID != "skill-a" {
		t.Fatalf("expected filtered skill skill-a, got %#v", result.Skills)
	}
}
