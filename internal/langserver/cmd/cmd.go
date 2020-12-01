package cmd

import (
	"context"
	"strings"
)

type Handler func(context.Context, CommandArgs) (interface{}, error)
type Handlers map[string]Handler

const langServerPrefix = "terraform-ls."

func Name(name string) string {
	return langServerPrefix + name
}

func (h Handlers) Names(commandPrefix string) (names []string) {
	if commandPrefix != "" {
		commandPrefix += "."
	}
	for name := range h {
		names = append(names, commandPrefix+name)
	}
	return names
}

func (h Handlers) Get(name, commandPrefix string) (Handler, bool) {
	if commandPrefix != "" {
		commandPrefix += "."
	}
	name = strings.TrimPrefix(name, commandPrefix)
	handler, ok := h[name]
	return handler, ok
}
