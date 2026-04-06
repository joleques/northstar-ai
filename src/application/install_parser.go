package application

import (
	"fmt"
	"strings"

	"github.com/joleques/northstar-ai/src/domain"
	usecase "github.com/joleques/northstar-ai/src/use_case"
)

type ParsedInstallCommand struct {
	Request usecase.InstallRequest
}

func ParseCLIArgs(args []string) (ParsedInstallCommand, error) {
	if len(args) == 0 {
		return ParsedInstallCommand{}, fmt.Errorf("missing command")
	}

	if args[0] != "install" {
		return ParsedInstallCommand{}, fmt.Errorf("unsupported command %q", args[0])
	}

	if len(args) >= 2 && (args[1] == "skills" || args[1] == "all") {
		return ParsedInstallCommand{}, fmt.Errorf("commands 'install skills' and 'install all' were removed; use 'install <assistant-id>'")
	}

	policy := domain.DefaultAgentsPolicy
	force := false
	outputDir := ""
	category := ""

	if len(args) >= 2 && args[1] == "assistant" {

		target, consumedTargetArgs, err := parseOptionalTarget(args[2:])
		if err != nil {
			return ParsedInstallCommand{}, err
		}

		assistants, options, err := parseAssistantsAndOptions(args[2+consumedTargetArgs:])
		if err != nil {
			return ParsedInstallCommand{}, err
		}

		policy, force, outputDir, category, err = parseInstallOptions(options, policy, force, outputDir, category)
		if err != nil {
			return ParsedInstallCommand{}, err
		}

		return ParsedInstallCommand{Request: usecase.InstallRequest{
			Target:       target,
			Assistants:   assistants,
			Category:     category,
			AgentsPolicy: policy,
			Force:        force,
			OutputDir:    outputDir,
		}}, nil
	}

	target, consumedTargetArgs, err := parseOptionalTarget(args[1:])
	if err != nil {
		return ParsedInstallCommand{}, err
	}

	assistants, options, err := parseAssistantsAndOptions(args[1+consumedTargetArgs:])
	if err != nil {
		return ParsedInstallCommand{}, err
	}

	policy, force, outputDir, category, err = parseInstallOptions(options, policy, force, outputDir, category)
	if err != nil {
		return ParsedInstallCommand{}, err
	}

	return ParsedInstallCommand{Request: usecase.InstallRequest{
		Target:       target,
		Assistants:   assistants,
		Category:     category,
		AgentsPolicy: policy,
		Force:        force,
		OutputDir:    outputDir,
	}}, nil
}

func parseOptionalTarget(values []string) (domain.TargetPlatform, int, error) {
	if len(values) == 0 {
		return "", 0, nil
	}

	first := strings.TrimSpace(values[0])
	if strings.HasPrefix(first, "-") {
		return "", 0, nil
	}

	target, err := domain.ParseTargetPlatform(first)
	if err == nil {
		return target, 1, nil
	}

	return "", 0, nil
}

func parseAssistantsAndOptions(values []string) ([]string, []string, error) {
	assistants := make([]string, 0)
	options := make([]string, 0)

	for i := 0; i < len(values); i++ {
		current := strings.TrimSpace(values[i])
		if current == "" {
			continue
		}

		if strings.HasPrefix(current, "--") {
			switch {
			case current == "--force":
				options = append(options, current)
			case strings.HasPrefix(current, "--category="):
				options = append(options, current)
			case current == "--category":
				if i+1 >= len(values) {
					return nil, nil, fmt.Errorf("missing value for --category")
				}
				next := strings.TrimSpace(values[i+1])
				if next == "" || strings.HasPrefix(next, "--") {
					return nil, nil, fmt.Errorf("missing value for --category")
				}
				options = append(options, current, next)
				i++
			case strings.HasPrefix(current, "--agents-policy="):
				options = append(options, current)
			case current == "--agents-policy":
				if i+1 >= len(values) {
					return nil, nil, fmt.Errorf("missing value for --agents-policy")
				}
				next := strings.TrimSpace(values[i+1])
				if next == "" || strings.HasPrefix(next, "--") {
					return nil, nil, fmt.Errorf("missing value for --agents-policy")
				}
				options = append(options, current, next)
				i++
			case strings.HasPrefix(current, "--output="):
				options = append(options, current)
			case current == "--output":
				if i+1 >= len(values) {
					return nil, nil, fmt.Errorf("missing value for --output")
				}
				next := strings.TrimSpace(values[i+1])
				if next == "" || strings.HasPrefix(next, "--") {
					return nil, nil, fmt.Errorf("missing value for --output")
				}
				options = append(options, current, next)
				i++
			default:
				return nil, nil, fmt.Errorf("unknown option %q", current)
			}
			continue
		}

		if strings.HasPrefix(current, "-") {
			return nil, nil, fmt.Errorf("unknown option %q", current)
		}

		assistants = append(assistants, current)
	}

	return assistants, options, nil
}

func parseInstallOptions(options []string, policy domain.AgentsPolicy, force bool, outputDir, category string) (domain.AgentsPolicy, bool, string, string, error) {
	categorySet := false

	for i := 0; i < len(options); i++ {
		current := options[i]
		switch {
		case current == "--force":
			force = true
		case strings.HasPrefix(current, "--category="):
			categorySet = true
			category = strings.TrimSpace(strings.TrimPrefix(current, "--category="))
		case current == "--category":
			i++
			categorySet = true
			category = strings.TrimSpace(options[i])
		case strings.HasPrefix(current, "--agents-policy="):
			value := strings.TrimSpace(strings.TrimPrefix(current, "--agents-policy="))
			policy = domain.AgentsPolicy(value)
		case current == "--agents-policy":
			i++
			policy = domain.AgentsPolicy(strings.TrimSpace(options[i]))
		case strings.HasPrefix(current, "--output="):
			outputDir = strings.TrimSpace(strings.TrimPrefix(current, "--output="))
		case current == "--output":
			i++
			outputDir = strings.TrimSpace(options[i])
		}
	}

	if err := policy.Validate(); err != nil {
		return "", false, "", "", err
	}

	if categorySet && (category == "" || strings.HasPrefix(category, "-")) {
		return "", false, "", "", fmt.Errorf("missing value for --category")
	}

	return policy, force, outputDir, category, nil
}
