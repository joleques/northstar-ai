package domain_test

import (
	"strings"
	"testing"

	"github.com/joleques/northstar-ai/src/domain"
)

func TestAssistantSpecValidateSuccess(t *testing.T) {
	t.Parallel()

	spec := domain.AssistantSpec{
		ID:           "code-reviewer",
		Name:         "Code Reviewer",
		Description:  "Revisa mudanças com foco em riscos.",
		Instructions: "Voce revisa codigo com foco em bugs, regressao e testes faltantes.",
		Inputs: []domain.InputSpec{
			{Name: "diff", Description: "Diff a ser revisado.", Required: true},
		},
		Tools: []string{"shell", "rg"},
		Tags:  []string{"quality", "review"},
	}

	if err := spec.Validate(); err != nil {
		t.Fatalf("expected valid assistant spec, got error: %v", err)
	}
}

func TestAssistantSpecValidateFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		spec        domain.AssistantSpec
		errContains string
	}{
		{
			name: "invalid id",
			spec: domain.AssistantSpec{
				ID:           "Code_Reviewer",
				Name:         "Code Reviewer",
				Description:  "desc",
				Instructions: strings.Repeat("a", 35),
			},
			errContains: "id must match",
		},
		{
			name: "short instructions",
			spec: domain.AssistantSpec{
				ID:           "code-reviewer",
				Name:         "Code Reviewer",
				Description:  "desc",
				Instructions: "curta demais",
			},
			errContains: "at least 30",
		},
		{
			name: "duplicate input name",
			spec: domain.AssistantSpec{
				ID:           "code-reviewer",
				Name:         "Code Reviewer",
				Description:  "desc",
				Instructions: strings.Repeat("a", 35),
				Inputs: []domain.InputSpec{
					{Name: "diff", Description: "d1"},
					{Name: "diff", Description: "d2"},
				},
			},
			errContains: "duplicated",
		},
		{
			name: "duplicate skills",
			spec: domain.AssistantSpec{
				ID:           "code-reviewer",
				Name:         "Code Reviewer",
				Description:  "desc",
				Instructions: strings.Repeat("a", 35),
				Skills:       []string{"researcher", "researcher"},
			},
			errContains: "skills",
		},
		{
			name: "duplicate tools",
			spec: domain.AssistantSpec{
				ID:           "code-reviewer",
				Name:         "Code Reviewer",
				Description:  "desc",
				Instructions: strings.Repeat("a", 35),
				Tools:        []string{"shell", "shell"},
			},
			errContains: "tools",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.spec.Validate()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("expected error containing %q, got: %v", tt.errContains, err)
			}
		})
	}
}

func TestAssistantSpecNormalized(t *testing.T) {
	t.Parallel()

	normalized := domain.AssistantSpec{
		ID:           "  code-reviewer  ",
		Name:         "  Code Reviewer  ",
		Description:  "  Revisor tecnico.  ",
		Instructions: "  Instrucao suficientemente longa para validacao.  ",
		Inputs: []domain.InputSpec{
			{Name: " diff ", Description: " valor ", Default: " x "},
		},
		Tools:    []string{" shell", "shell", "rg "},
		Tags:     []string{" quality ", "quality", "review"},
		Metadata: nil,
	}.Normalized()

	if normalized.Version != "0.1.0" {
		t.Fatalf("expected default version 0.1.0, got %q", normalized.Version)
	}

	if len(normalized.Tools) != 2 {
		t.Fatalf("expected 2 unique tools, got %d", len(normalized.Tools))
	}

	if len(normalized.Tags) != 2 {
		t.Fatalf("expected 2 unique tags, got %d", len(normalized.Tags))
	}

	if normalized.Inputs[0].Name != "diff" {
		t.Fatalf("expected input name to be trimmed, got %q", normalized.Inputs[0].Name)
	}

	if normalized.Metadata == nil {
		t.Fatal("expected metadata map to be initialized")
	}
}

func TestAgentsPolicyValidate(t *testing.T) {
	t.Parallel()

	if err := domain.DefaultAgentsPolicy.Validate(); err != nil {
		t.Fatalf("expected default policy to be valid, got %v", err)
	}

	if err := domain.AgentsPolicy("chaos").Validate(); err == nil {
		t.Fatal("expected invalid policy to return error")
	}
}

func TestParseTargetPlatform(t *testing.T) {
	t.Parallel()

	if _, err := domain.ParseTargetPlatform("codex"); err != nil {
		t.Fatalf("expected codex platform to be valid, got %v", err)
	}

	if _, err := domain.ParseTargetPlatform("unknown"); err == nil {
		t.Fatal("expected unknown platform to return error")
	}
}
