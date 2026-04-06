package application_test

import (
	"testing"

	"github.com/joleques/northstar-ai/src/application"
)

func TestParseListLibraryArgs(t *testing.T) {
	t.Parallel()

	parsed, err := application.ParseListLibraryArgs([]string{"list-lib"})
	if err != nil {
		t.Fatalf("expected valid list-lib args, got %v", err)
	}
	if parsed.Request.IncludeSkills {
		t.Fatal("expected include skills disabled by default")
	}

	parsed, err = application.ParseListLibraryArgs([]string{"list-lib", "--skills"})
	if err != nil {
		t.Fatalf("expected valid list-lib --skills args, got %v", err)
	}
	if !parsed.Request.IncludeSkills {
		t.Fatal("expected include skills enabled with --skills")
	}

	parsed, err = application.ParseListLibraryArgs([]string{"list-lib", "--category", "platform"})
	if err != nil {
		t.Fatalf("expected valid list-lib --category args, got %v", err)
	}
	if parsed.Request.Category != "platform" {
		t.Fatalf("expected category platform, got %q", parsed.Request.Category)
	}

	parsed, err = application.ParseListLibraryArgs([]string{"list-lib", "--skills", "--category=documentation"})
	if err != nil {
		t.Fatalf("expected valid list-lib with --skills and --category args, got %v", err)
	}
	if parsed.Request.Category != "documentation" {
		t.Fatalf("expected category documentation, got %q", parsed.Request.Category)
	}

	parsed, err = application.ParseListLibraryArgs([]string{"list-lib", "--output", "/tmp/client"})
	if err != nil {
		t.Fatalf("expected valid list-lib --output args, got %v", err)
	}
	if parsed.Request.OutputDir != "/tmp/client" {
		t.Fatalf("expected output /tmp/client, got %q", parsed.Request.OutputDir)
	}

	if _, err := application.ParseListLibraryArgs([]string{"list-lib", "--wat"}); err == nil {
		t.Fatal("expected invalid list-lib args to return error")
	}

	if _, err := application.ParseListLibraryArgs([]string{"list-lib", "--category"}); err == nil {
		t.Fatal("expected missing --category value to return error")
	}

	if _, err := application.ParseListLibraryArgs([]string{"list-lib", "--output"}); err == nil {
		t.Fatal("expected missing --output value to return error")
	}
}
