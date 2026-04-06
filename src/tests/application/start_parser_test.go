package application_test

import (
	"strings"
	"testing"

	"github.com/joleques/northstar-ai/src/application"
)

func TestParseStartArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		input          string
		expectTarget   string
		expectTitle    string
		expectDesc     string
		expectDocs     []string
		expectOutput   string
		expectForce    bool
		expectInteract bool
		wantErr        bool
	}{
		{
			name:         "non interactive flags",
			args:         []string{"start", "--target", "codex", "--title", "Heimdall", "--description", "Project context", "--doc", "README.md", "--doc", "docs/vision.md", "--output", "/tmp/project", "--force"},
			expectTarget: "codex",
			expectTitle:  "Heimdall",
			expectDesc:   "Project context",
			expectDocs:   []string{"README.md", "docs/vision.md"},
			expectOutput: "/tmp/project",
			expectForce:  true,
		},
		{
			name:           "interactive fallback",
			args:           []string{"start"},
			input:          "Heimdall\nProject context\n",
			expectTarget:   "",
			expectTitle:    "Heimdall",
			expectDesc:     "Project context",
			expectDocs:     []string{},
			expectInteract: false,
		},
		{
			name:           "interactive flag preserved",
			args:           []string{"start", "--interactive", "--target", "codex", "--title", "Heimdall", "--description", "Project context", "--doc", "README.md"},
			expectTarget:   "codex",
			expectTitle:    "Heimdall",
			expectDesc:     "Project context",
			expectDocs:     []string{"README.md"},
			expectInteract: true,
		},
		{
			name:    "unknown option",
			args:    []string{"start", "--wat"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var reader *strings.Reader
			if tt.input != "" {
				reader = strings.NewReader(tt.input)
			}

			got, err := application.ParseStartArgs(tt.args, reader)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if got.Request.Title != tt.expectTitle {
				t.Fatalf("expected title %q, got %q", tt.expectTitle, got.Request.Title)
			}

			if got.Request.Description != tt.expectDesc {
				t.Fatalf("expected description %q, got %q", tt.expectDesc, got.Request.Description)
			}

			if string(got.Request.Target) != tt.expectTarget {
				t.Fatalf("expected target %q, got %q", tt.expectTarget, got.Request.Target)
			}

			if len(got.Request.Documentation) != len(tt.expectDocs) {
				t.Fatalf("expected %d docs, got %d", len(tt.expectDocs), len(got.Request.Documentation))
			}

			for index, doc := range tt.expectDocs {
				if got.Request.Documentation[index] != doc {
					t.Fatalf("expected doc %d to be %q, got %q", index, doc, got.Request.Documentation[index])
				}
			}

			if got.Request.OutputDir != tt.expectOutput {
				t.Fatalf("expected output %q, got %q", tt.expectOutput, got.Request.OutputDir)
			}

			if got.Request.Force != tt.expectForce {
				t.Fatalf("expected force %v, got %v", tt.expectForce, got.Request.Force)
			}

			if got.Interactive != tt.expectInteract {
				t.Fatalf("expected interactive %v, got %v", tt.expectInteract, got.Interactive)
			}
		})
	}
}
