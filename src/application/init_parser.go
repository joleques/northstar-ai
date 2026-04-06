package application

import (
	"fmt"
	"strings"

	"github.com/joleques/northstar-ai/src/domain"
	usecase "github.com/joleques/northstar-ai/src/use_case"
)

type ParsedInitCommand struct {
	Request usecase.InitRequest
}

func ParseInitArgs(args []string) (ParsedInitCommand, error) {
	if len(args) == 0 {
		return ParsedInitCommand{}, fmt.Errorf("missing command")
	}

	if args[0] != "init" {
		return ParsedInitCommand{}, fmt.Errorf("unsupported command %q", args[0])
	}

	if len(args) < 2 {
		return ParsedInitCommand{}, fmt.Errorf("usage: init <target> [--agents-policy <skip|if-missing|overwrite>] [--force] [--output <dir>]")
	}

	target, err := domain.ParseTargetPlatform(args[1])
	if err != nil {
		return ParsedInitCommand{}, err
	}

	policy := domain.DefaultAgentsPolicy
	force := false
	outputDir := ""

	for i := 2; i < len(args); i++ {
		current := strings.TrimSpace(args[i])
		switch {
		case current == "--force":
			force = true
		case strings.HasPrefix(current, "--agents-policy="):
			policy = domain.AgentsPolicy(strings.TrimSpace(strings.TrimPrefix(current, "--agents-policy=")))
		case current == "--agents-policy":
			if i+1 >= len(args) {
				return ParsedInitCommand{}, fmt.Errorf("missing value for --agents-policy")
			}
			i++
			policy = domain.AgentsPolicy(strings.TrimSpace(args[i]))
		case strings.HasPrefix(current, "--output="):
			outputDir = strings.TrimSpace(strings.TrimPrefix(current, "--output="))
		case current == "--output":
			if i+1 >= len(args) {
				return ParsedInitCommand{}, fmt.Errorf("missing value for --output")
			}
			i++
			outputDir = strings.TrimSpace(args[i])
		default:
			return ParsedInitCommand{}, fmt.Errorf("unknown option %q", current)
		}
	}

	if err := policy.Validate(); err != nil {
		return ParsedInitCommand{}, err
	}

	return ParsedInitCommand{Request: usecase.InitRequest{
		Target:       target,
		AgentsPolicy: policy,
		Force:        force,
		OutputDir:    outputDir,
	}}, nil
}
