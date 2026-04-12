package infra_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joleques/northstar-ai/src/domain"
	infrainstall "github.com/joleques/northstar-ai/src/infra/install"
	usecase "github.com/joleques/northstar-ai/src/use_case"
)

func TestFilesystemGatewayUpdateAppReinstallsPlatformAssistantWrappers(t *testing.T) {
	t.Parallel()

	templateRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(templateRoot, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateRoot, "AGENTS.md"), []byte("agents-template"), 0o644); err != nil {
		t.Fatal(err)
	}

	currentPlatformAssistant := `type: assitent
categories:
  - platform
id: northstar-start
name: Northstar Start
description: Start helper.
instructions: |
  execute start helper
skills: []`
	if err := os.WriteFile(filepath.Join(templateRoot, "tools", "northstar-start.yaml"), []byte(currentPlatformAssistant), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outputDir, ".codex", "skills", "northstar-start"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, ".codex", "skills", "northstar-start", "SKILL.md"), []byte("legacy wrapper"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(outputDir, ".northstar", "template", "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	previousPlatformAssistant := `type: assitent
categories:
  - platform
id: northstar-start
name: Northstar Start
description: Start helper v1.
instructions: |
  execute old start helper
skills: []`
	if err := os.WriteFile(filepath.Join(outputDir, ".northstar", "template", "tools", "northstar-start.yaml"), []byte(previousPlatformAssistant), 0o644); err != nil {
		t.Fatal(err)
	}

	gateway := infrainstall.NewFilesystemGateway(filepath.Join(templateRoot, "AGENTS.md"))
	result, err := gateway.UpdateApp(context.Background(), usecase.UpdateAppRequest{
		Target:    domain.TargetCodex,
		OutputDir: outputDir,
	})
	if err != nil {
		t.Fatalf("expected update-app to succeed, got %v", err)
	}

	wrapperPath := filepath.Join(outputDir, ".codex", "skills", "northstar-start", "SKILL.md")
	content, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("expected current platform assistant wrapper to be reinstalled, got %v", err)
	}
	if len(content) == 0 {
		t.Fatalf("expected wrapper content to be regenerated, got empty file")
	}
	if len(result.Installed) == 0 {
		t.Fatalf("expected installed entries, got %#v", result)
	}
}
