package application

import (
	"fmt"
	"strings"

	usecase "github.com/joleques/northstar-ai/src/use_case"
)

type ParsedUpdateAppCommand struct {
	Request usecase.UpdateAppRequest
}

func ParseUpdateAppArgs(args []string) (ParsedUpdateAppCommand, error) {
	if len(args) == 0 {
		return ParsedUpdateAppCommand{}, fmt.Errorf("missing command")
	}

	if args[0] != "update-app" {
		return ParsedUpdateAppCommand{}, fmt.Errorf("unsupported command %q", args[0])
	}

	target, consumedTargetArgs, err := parseOptionalTarget(args[1:])
	if err != nil {
		return ParsedUpdateAppCommand{}, err
	}

	outputDir := ""
	options := args[1+consumedTargetArgs:]
	for i := 0; i < len(options); i++ {
		current := strings.TrimSpace(options[i])
		if current == "" {
			continue
		}

		switch {
		case strings.HasPrefix(current, "--output="):
			outputDir = strings.TrimSpace(strings.TrimPrefix(current, "--output="))
		case current == "--output":
			if i+1 >= len(options) {
				return ParsedUpdateAppCommand{}, fmt.Errorf("missing value for --output")
			}
			next := strings.TrimSpace(options[i+1])
			if next == "" || strings.HasPrefix(next, "--") {
				return ParsedUpdateAppCommand{}, fmt.Errorf("missing value for --output")
			}
			outputDir = next
			i++
		default:
			return ParsedUpdateAppCommand{}, fmt.Errorf("unknown option %q", current)
		}
	}

	return ParsedUpdateAppCommand{
		Request: usecase.UpdateAppRequest{
			Target:    target,
			OutputDir: outputDir,
		},
	}, nil
}
