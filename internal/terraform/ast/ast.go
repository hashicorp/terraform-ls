// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ast

import "strings"

// TODO move this to a more appropriate place
type RecordType int64

const (
	RecordTypeModule RecordType = iota
	RecordTypeVariable
	RecordTypeRoot
)

func (rt RecordType) String() string {
	switch rt {
	case RecordTypeModule:
		return "module"
	case RecordTypeVariable:
		return "variable"
	case RecordTypeRoot:
		return "root"
	default:
		return "unknown"
	}
}

func RecordTypeFromLanguageID(languageID string) RecordType {
	switch languageID {
	case "terraform":
		return RecordTypeModule
	case "terraform-vars":
		return RecordTypeVariable
	default:
		return -1 // TODO!
	}
}

func IsSupportedFilename(name string) bool {
	return IsModuleFilename(name) || IsVarsFilename(name)
}

// isIgnoredFile returns true if the given filename (which must not have a
// directory path ahead of it) should be ignored as e.g. an editor swap file.
// See https://github.com/hashicorp/terraform/blob/d35bc05/internal/configs/parser_config_dir.go#L107
func IsIgnoredFile(name string) bool {
	return strings.HasPrefix(name, ".") || // Unix-like hidden files
		strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#") // emacs
}
