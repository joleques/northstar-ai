package application_test

import (
	"reflect"
	"testing"

	"github.com/joleques/northstar-ai/src/application"
	"github.com/joleques/northstar-ai/src/domain"
)

func TestParseCLIArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		expectTarget domain.TargetPlatform
		expectAssts  []string
		expectPolicy domain.AgentsPolicy
		expectForce  bool
		expectOutput string
		expectCat    string
		wantErr      bool
	}{
		{
			name:         "context based install command",
			args:         []string{"install", "assistant-a"},
			expectTarget: "",
			expectAssts:  []string{"assistant-a"},
			expectPolicy: domain.DefaultAgentsPolicy,
		},
		{
			name:         "install by skill id",
			args:         []string{"install", "skill-a"},
			expectTarget: "",
			expectAssts:  []string{"skill-a"},
			expectPolicy: domain.DefaultAgentsPolicy,
		},
		{
			name:         "install all with no filters",
			args:         []string{"install"},
			expectTarget: "",
			expectAssts:  []string{},
			expectPolicy: domain.DefaultAgentsPolicy,
		},
		{
			name:         "assistant command",
			args:         []string{"install", "assistant", "assistant-a"},
			expectTarget: "",
			expectAssts:  []string{"assistant-a"},
			expectPolicy: domain.DefaultAgentsPolicy,
		},
		{
			name:         "legacy assistant command with target",
			args:         []string{"install", "assistant", "codex", "assistant-a"},
			expectTarget: domain.TargetCodex,
			expectAssts:  []string{"assistant-a"},
			expectPolicy: domain.DefaultAgentsPolicy,
		},
		{
			name:         "target alias command",
			args:         []string{"install", "claude", "assistant-a", "assistant-b"},
			expectTarget: domain.TargetClaude,
			expectAssts:  []string{"assistant-a", "assistant-b"},
			expectPolicy: domain.DefaultAgentsPolicy,
		},
		{
			name:         "options",
			args:         []string{"install", "assistant", "cursor", "assistant-a", "--agents-policy=overwrite", "--force", "--output", "/tmp/work"},
			expectTarget: domain.TargetCursor,
			expectAssts:  []string{"assistant-a"},
			expectPolicy: domain.AgentsPolicyOverwrite,
			expectForce:  true,
			expectOutput: "/tmp/work",
		},
		{
			name:         "install by category",
			args:         []string{"install", "--category", "documentation"},
			expectTarget: "",
			expectAssts:  []string{},
			expectPolicy: domain.DefaultAgentsPolicy,
			expectCat:    "documentation",
		},
		{
			name:         "install assistant by category with target",
			args:         []string{"install", "assistant", "codex", "--category=media"},
			expectTarget: domain.TargetCodex,
			expectAssts:  []string{},
			expectPolicy: domain.DefaultAgentsPolicy,
			expectCat:    "media",
		},
		{
			name:    "legacy skills removed",
			args:    []string{"install", "skills", "codex", "skill-a"},
			wantErr: true,
		},
		{
			name:         "legacy target only means install all",
			args:         []string{"install", "assistant", "codex"},
			expectTarget: domain.TargetCodex,
			expectAssts:  []string{},
			expectPolicy: domain.DefaultAgentsPolicy,
		},
		{
			name:    "missing category value",
			args:    []string{"install", "--category"},
			wantErr: true,
		},
		{
			name:    "unknown option",
			args:    []string{"install", "assistant", "codex", "assistant-a", "--wat"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := application.ParseCLIArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if got.Request.Target != tt.expectTarget {
				t.Fatalf("expected target %q, got %q", tt.expectTarget, got.Request.Target)
			}

			if got.Request.AgentsPolicy != tt.expectPolicy {
				t.Fatalf("expected policy %q, got %q", tt.expectPolicy, got.Request.AgentsPolicy)
			}

			if got.Request.Force != tt.expectForce {
				t.Fatalf("expected force %v, got %v", tt.expectForce, got.Request.Force)
			}

			if got.Request.OutputDir != tt.expectOutput {
				t.Fatalf("expected output %q, got %q", tt.expectOutput, got.Request.OutputDir)
			}

			if got.Request.Category != tt.expectCat {
				t.Fatalf("expected category %q, got %q", tt.expectCat, got.Request.Category)
			}

			if !reflect.DeepEqual(got.Request.Assistants, tt.expectAssts) {
				t.Fatalf("expected assistants %v, got %v", tt.expectAssts, got.Request.Assistants)
			}
		})
	}
}
