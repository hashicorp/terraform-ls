// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package lsp

// LanguageID represents the coding language
// of a file
type LanguageID string

const (
	Terraform LanguageID = "terraform"
	Tfvars    LanguageID = "terraform-vars"
	Stacks    LanguageID = "terraform-stack"
	Deploy    LanguageID = "terraform-deploy"
	Test      LanguageID = "terraform-test"
	Mock      LanguageID = "terraform-mock"
	Search    LanguageID = "terraform-search"
)

func (l LanguageID) String() string {
	return string(l)
}
