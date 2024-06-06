// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ast

func IsRootModuleFilename(name string) bool {
	return (name == ".terraform.lock.hcl" ||
		name == ".terraform-version")
}
