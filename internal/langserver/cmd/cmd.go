// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"sort"
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

	sort.SliceStable(names, func(i, j int) bool {
		return names[i] < names[j]
	})

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
