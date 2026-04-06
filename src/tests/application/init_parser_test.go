package application_test

import (
	"testing"

	"github.com/joleques/northstar-ai/src/application"
	"github.com/joleques/northstar-ai/src/domain"
)

func TestParseInitArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		expectTarget domain.TargetPlatform
		expectPolicy domain.AgentsPolicy
		expectForce  bool
		expectOutput string
		wantErr      bool
	}{
		{
			name:         "basic init",
			args:         []string{"init", "codex"},
			expectTarget: domain.TargetCodex,
			expectPolicy: domain.DefaultAgentsPolicy,
		},
		{
			name:         "init with options",
			args:         []string{"init", "cursor", "--agents-policy", "overwrite", "--force", "--output", "/tmp/work"},
			expectTarget: domain.TargetCursor,
			expectPolicy: domain.AgentsPolicyOverwrite,
			expectForce:  true,
			expectOutput: "/tmp/work",
		},
		{
			name:    "invalid command",
			args:    []string{"install", "codex"},
			wantErr: true,
		},
		{
			name:    "missing target",
			args:    []string{"init"},
			wantErr: true,
		},
		{
			name:    "unknown option",
			args:    []string{"init", "codex", "--wat"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := application.ParseInitArgs(tt.args)
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
		})
	}
}
