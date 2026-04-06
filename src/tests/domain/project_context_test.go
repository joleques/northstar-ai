package domain_test

import (
	"strings"
	"testing"

	"github.com/joleques/northstar-ai/src/domain"
)

func TestProjectContextValidate(t *testing.T) {
	t.Parallel()

	context := domain.ProjectContext{
		Target:        domain.TargetCodex,
		Title:         "Heimdall",
		Description:   "Coordena assistants com contexto canônico.",
		Documentation: []string{"README.md"},
	}

	if err := context.Validate(); err != nil {
		t.Fatalf("expected valid project context, got %v", err)
	}
}

func TestProjectContextValidateAllowsEmptyDocumentation(t *testing.T) {
	t.Parallel()

	context := domain.ProjectContext{
		Target:      domain.TargetCodex,
		Title:       "Heimdall",
		Description: "Coordena assistants com contexto canônico.",
	}

	if err := context.Validate(); err != nil {
		t.Fatalf("expected valid project context without docs, got %v", err)
	}
}

func TestProjectContextValidateFailures(t *testing.T) {
	t.Parallel()

	context := domain.ProjectContext{
		Target:        "",
		Title:         " ",
		Description:   " ",
		Documentation: []string{"", "README.md"},
	}

	err := context.Validate()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	for _, fragment := range []string{"target is required", "title is required", "description is required", "documentation[0] is required"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("expected error containing %q, got %v", fragment, err)
		}
	}
}

func TestProjectContextNormalized(t *testing.T) {
	t.Parallel()

	normalized := domain.ProjectContext{
		Target:        " codex ",
		ProjectRoot:   "  /tmp/heimdall  ",
		Title:         "  Heimdall App  ",
		Description:   "  Contexto principal.  ",
		Documentation: []string{" README.md ", "README.md", " docs/vision.md "},
	}.Normalized()

	if normalized.Target != domain.TargetCodex {
		t.Fatalf("expected trimmed target codex, got %q", normalized.Target)
	}

	if normalized.Title != "Heimdall App" {
		t.Fatalf("expected trimmed title, got %q", normalized.Title)
	}
	if normalized.ProjectRoot != "/tmp/heimdall" {
		t.Fatalf("expected trimmed project root, got %q", normalized.ProjectRoot)
	}

	if normalized.Description != "Contexto principal." {
		t.Fatalf("expected trimmed description, got %q", normalized.Description)
	}

	if len(normalized.Documentation) != 2 {
		t.Fatalf("expected 2 unique documentation entries, got %d", len(normalized.Documentation))
	}
}
