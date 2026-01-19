// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package ast

import "strings"

// isIgnoredFile returns true if the given filename (which must not have a
// directory path ahead of it) should be ignored as e.g. an editor swap file.
// See https://github.com/hashicorp/terraform/blob/d35bc05/internal/configs/parser_config_dir.go#L107
func IsIgnoredFile(name string) bool {
	return strings.HasPrefix(name, ".") || // Unix-like hidden files
		strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#") // emacs
}
