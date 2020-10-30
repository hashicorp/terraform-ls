package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/creachadair/jrpc2/code"
	lsp "github.com/sourcegraph/go-lsp"
)

type executeCommandHandler func(context.Context, commandArgs) (interface{}, error)
type executeCommandHandlers map[string]executeCommandHandler

const langServerPrefix = "terraform-ls."

var commandPrefix string
var handlers = make(executeCommandHandlers)

func prefix(name string) string {
	prefix := langServerPrefix
	if commandPrefix != "" {
		prefix = commandPrefix + "." + langServerPrefix
	}
	return prefix + name
}

func (h executeCommandHandlers) Init(p string) executeCommandHandlers {
	if len(h) == 0 {
		commandPrefix = p
		h[prefix("rootmodules")] = executeCommandRootModulesHandler
	}
	return h
}

func (h executeCommandHandlers) Names() (names []string) {
	for name := range h {
		names = append(names, name)
	}
	return names
}

func (lh *logHandler) WorkspaceExecuteCommand(ctx context.Context, params lsp.ExecuteCommandParams) (interface{}, error) {
	handler, ok := handlers[params.Command]
	if !ok {
		return nil, fmt.Errorf("%w: command handler not found for %q", code.MethodNotFound.Err(), params.Command)
	}
	return handler(ctx, parseCommandArgs(params.Arguments))
}

type commandArgs map[string]interface{}

func (c commandArgs) GetString(variable string) (string, bool) {
	vRaw, ok := c[variable]
	if !ok {
		return "", false
	}
	v, ok := vRaw.(string)
	if !ok {
		return "", false
	}
	return v, true
}

func (c commandArgs) GetNumber(variable string) (float64, bool) {
	vRaw, ok := c[variable]
	if !ok {
		return 0, false
	}
	v, ok := vRaw.(float64)
	if !ok {
		return 0, false
	}
	return v, true
}

func (c commandArgs) GetBool(variable string) (bool, bool) {
	vRaw, ok := c[variable]
	if !ok {
		return false, false
	}
	v, ok := vRaw.(bool)
	if !ok {
		return false, false
	}
	return v, true
}

func parseCommandArgs(arguments []interface{}) commandArgs {
	args := make(map[string]interface{})
	if arguments == nil {
		return args
	}
	for _, rawArg := range arguments {
		arg, ok := rawArg.(string)
		if !ok {
			continue
		}
		if arg == "" {
			continue
		}

		pair := strings.SplitN(arg, "=", 2)
		if len(pair) != 2 {
			continue
		}

		variable := strings.ToLower(pair[0])
		value := pair[1]
		if value == "" {
			args[variable] = value
			continue
		}

		if f, err := strconv.ParseFloat(value, 64); err == nil {
			args[variable] = f
		} else if b, err := strconv.ParseBool(value); err == nil {
			args[variable] = b
		} else {
			args[variable] = value
		}

	}
	return args
}
