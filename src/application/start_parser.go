package application

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/joleques/northstar-ai/src/domain"
	usecase "github.com/joleques/northstar-ai/src/use_case"
)

type ParsedStartCommand struct {
	Request     usecase.StartRequest
	Interactive bool
}

func ParseStartArgs(args []string, in io.Reader) (ParsedStartCommand, error) {
	if len(args) == 0 {
		return ParsedStartCommand{}, fmt.Errorf("missing command")
	}

	if args[0] != "start" {
		return ParsedStartCommand{}, fmt.Errorf("unsupported command %q", args[0])
	}

	request := usecase.StartRequest{
		Documentation: []string{},
	}
	interactive := false

	for i := 1; i < len(args); i++ {
		current := strings.TrimSpace(args[i])
		switch {
		case current == "--force":
			request.Force = true
		case current == "--interactive":
			interactive = true
		case strings.HasPrefix(current, "--target="):
			target, err := domain.ParseTargetPlatform(strings.TrimSpace(strings.TrimPrefix(current, "--target=")))
			if err != nil {
				return ParsedStartCommand{}, err
			}
			request.Target = target
		case current == "--target":
			if i+1 >= len(args) {
				return ParsedStartCommand{}, fmt.Errorf("missing value for --target")
			}
			i++
			target, err := domain.ParseTargetPlatform(strings.TrimSpace(args[i]))
			if err != nil {
				return ParsedStartCommand{}, err
			}
			request.Target = target
		case strings.HasPrefix(current, "--title="):
			request.Title = strings.TrimSpace(strings.TrimPrefix(current, "--title="))
		case current == "--title":
			if i+1 >= len(args) {
				return ParsedStartCommand{}, fmt.Errorf("missing value for --title")
			}
			i++
			request.Title = strings.TrimSpace(args[i])
		case strings.HasPrefix(current, "--description="):
			request.Description = strings.TrimSpace(strings.TrimPrefix(current, "--description="))
		case current == "--description":
			if i+1 >= len(args) {
				return ParsedStartCommand{}, fmt.Errorf("missing value for --description")
			}
			i++
			request.Description = strings.TrimSpace(args[i])
		case strings.HasPrefix(current, "--doc="):
			request.Documentation = append(request.Documentation, strings.TrimSpace(strings.TrimPrefix(current, "--doc=")))
		case current == "--doc":
			if i+1 >= len(args) {
				return ParsedStartCommand{}, fmt.Errorf("missing value for --doc")
			}
			i++
			request.Documentation = append(request.Documentation, strings.TrimSpace(args[i]))
		case strings.HasPrefix(current, "--output="):
			request.OutputDir = strings.TrimSpace(strings.TrimPrefix(current, "--output="))
		case current == "--output":
			if i+1 >= len(args) {
				return ParsedStartCommand{}, fmt.Errorf("missing value for --output")
			}
			i++
			request.OutputDir = strings.TrimSpace(args[i])
		default:
			return ParsedStartCommand{}, fmt.Errorf("unknown option %q", current)
		}
	}

	if in != nil && (interactive || needsInteractiveStartInput(request)) {
		var err error
		request, err = collectStartInput(in, request)
		if err != nil {
			return ParsedStartCommand{}, err
		}
	}

	return ParsedStartCommand{Request: request, Interactive: interactive}, nil
}

func needsInteractiveStartInput(request usecase.StartRequest) bool {
	return strings.TrimSpace(request.Title) == "" ||
		strings.TrimSpace(request.Description) == ""
}

func collectStartInput(in io.Reader, request usecase.StartRequest) (usecase.StartRequest, error) {
	if in == nil {
		return request, fmt.Errorf("missing input reader for interactive start")
	}

	reader := bufio.NewReader(in)
	var err error

	if strings.TrimSpace(request.Title) == "" {
		request.Title, err = readLine(reader)
		if err != nil {
			return request, fmt.Errorf("read title: %w", err)
		}
	}

	if strings.TrimSpace(request.Description) == "" {
		request.Description, err = readLine(reader)
		if err != nil {
			return request, fmt.Errorf("read description: %w", err)
		}
	}

	return request, nil
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	if err == io.EOF && line == "" {
		return "", io.EOF
	}

	return strings.TrimSpace(line), nil
}

func splitDocumentationEntries(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized := strings.TrimSpace(part)
		if normalized == "" {
			continue
		}
		result = append(result, normalized)
	}
	return result
}
