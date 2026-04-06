package application

import (
	"fmt"
	"strings"

	usecase "github.com/joleques/northstar-ai/src/use_case"
)

type ParsedListLibraryCommand struct {
	Request usecase.ListLibraryRequest
}

func ParseListLibraryArgs(args []string) (ParsedListLibraryCommand, error) {
	if len(args) == 0 {
		return ParsedListLibraryCommand{}, fmt.Errorf("missing command")
	}

	if args[0] != "list-lib" {
		return ParsedListLibraryCommand{}, fmt.Errorf("unsupported command %q", args[0])
	}

	request := usecase.ListLibraryRequest{}
	for i := 1; i < len(args); i++ {
		option := strings.TrimSpace(args[i])
		switch option {
		case "--skills":
			request.IncludeSkills = true
		case "--category":
			if i+1 >= len(args) {
				return ParsedListLibraryCommand{}, fmt.Errorf("missing value for --category")
			}
			i++
			request.Category = strings.TrimSpace(args[i])
		case "--output":
			if i+1 >= len(args) {
				return ParsedListLibraryCommand{}, fmt.Errorf("missing value for --output")
			}
			i++
			request.OutputDir = strings.TrimSpace(args[i])
		default:
			if strings.HasPrefix(option, "--category=") {
				request.Category = strings.TrimSpace(strings.TrimPrefix(option, "--category="))
				break
			}
			if strings.HasPrefix(option, "--output=") {
				request.OutputDir = strings.TrimSpace(strings.TrimPrefix(option, "--output="))
				break
			}
			return ParsedListLibraryCommand{}, fmt.Errorf("unknown option %q", option)
		}
	}

	if request.Category == "" && len(args) > 1 {
		for _, arg := range args[1:] {
			if strings.TrimSpace(arg) == "--category" || strings.HasPrefix(strings.TrimSpace(arg), "--category=") {
				return ParsedListLibraryCommand{}, fmt.Errorf("missing value for --category")
			}
		}
	}
	if request.OutputDir == "" && len(args) > 1 {
		for _, arg := range args[1:] {
			if strings.TrimSpace(arg) == "--output" || strings.HasPrefix(strings.TrimSpace(arg), "--output=") {
				return ParsedListLibraryCommand{}, fmt.Errorf("missing value for --output")
			}
		}
	}

	return ParsedListLibraryCommand{Request: request}, nil
}
