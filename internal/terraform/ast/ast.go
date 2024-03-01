// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ast

func IsSupportedFilename(name string) bool {
	return IsModuleFilename(name) || IsVarsFilename(name)
}
