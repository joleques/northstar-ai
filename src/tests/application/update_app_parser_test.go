package application_test

import (
	"testing"

	"github.com/heimdall-app/heimdall/src/application"
	"github.com/heimdall-app/heimdall/src/domain"
)

func TestParseUpdateAppArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		expectTarget domain.TargetPlatform
		expectOutput string
		wantErr      bool
	}{
		{
			name:         "command without options",
			args:         []string{"update-app"},
			expectTarget: "",
		},
		{
			name:         "command with explicit target and output",
			args:         []string{"update-app", "codex", "--output", "/tmp/work"},
			expectTarget: domain.TargetCodex,
			expectOutput: "/tmp/work",
		},
		{
			name:         "command with output equals",
			args:         []string{"update-app", "--output=/tmp/work"},
			expectTarget: "",
			expectOutput: "/tmp/work",
		},
		{
			name:    "missing output value",
			args:    []string{"update-app", "--output"},
			wantErr: true,
		},
		{
			name:    "unknown option",
			args:    []string{"update-app", "--force"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := application.ParseUpdateAppArgs(tt.args)
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
			if got.Request.OutputDir != tt.expectOutput {
				t.Fatalf("expected output %q, got %q", tt.expectOutput, got.Request.OutputDir)
			}
		})
	}
}
