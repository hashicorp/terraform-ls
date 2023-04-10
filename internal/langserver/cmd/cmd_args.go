// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"encoding/json"
	"strconv"
	"strings"
)

type CommandArgs map[string]interface{}

func (c CommandArgs) GetString(variable string) (string, bool) {
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

func (c CommandArgs) GetNumber(variable string) (float64, bool) {
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

func (c CommandArgs) GetBool(variable string) (bool, bool) {
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

func ParseCommandArgs(arguments []json.RawMessage) CommandArgs {
	args := make(map[string]interface{})
	if arguments == nil {
		return args
	}
	for _, rawArg := range arguments {
		var arg string
		err := json.Unmarshal(rawArg, &arg)
		if err != nil {
			// TODO: Log error
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
