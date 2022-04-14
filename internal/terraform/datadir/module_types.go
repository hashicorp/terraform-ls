package datadir

import (
	"strings"

	tfregistry "github.com/hashicorp/terraform-registry-address"
)

type ModuleType string

const (
	UNKNOWN    ModuleType = "unknown"
	TFREGISTRY ModuleType = "tfregistry"
	LOCAL      ModuleType = "local"
	GITHUB     ModuleType = "github"
	GIT        ModuleType = "git"
)

var moduleSourceLocalPrefixes = []string{
	"./",
	"../",
	".\\",
	"..\\",
}

// GetModuleType parses source addresses to determine what kind of source the Terraform module comes
// from. It currently supports detecting Terraform Registry modules, GitHub modules, Git modules, and
// local file paths
func (r *ModuleRecord) GetModuleType() ModuleType {
	// TODO: It is technically incorrect to use the package hashicorp/terraform-registry-address
	// here as it is written to parse Terraform provider addresses and may not work correctly on
	// Terraform module addresses. The proper approach is to create a new parsing library that is
	// dedicated to parsing these kinds of addresses correctly, by re-using the logic defined in
	// the authorative source: hashicorp/terraform/internal/addrs/module_source.go.
	// However this works enough for now to identify module types for display in vscode-terraform.
	// Example: terraform-aws-modules/ec2-instance/aws
	_, err := tfregistry.ParseRawProviderSourceString(r.SourceAddr)
	// Example: registry.terraform.io/terraform-aws-modules/vpc/aws
	if err == nil || strings.HasPrefix(r.SourceAddr, "registry.terraform.io/") {
		return TFREGISTRY
	}

	// Example: github.com/terraform-aws-modules/terraform-aws-security-group
	if strings.HasPrefix(r.SourceAddr, "github.com/") {
		return GITHUB
	}

	// Example: git::https://example.com/vpc.git
	if strings.HasPrefix(r.SourceAddr, "git::") {
		return GIT
	}

	// Local file paths
	if isModuleSourceLocal(r.SourceAddr) {
		return LOCAL
	}

	return UNKNOWN
}

func isModuleSourceLocal(raw string) bool {
	for _, prefix := range moduleSourceLocalPrefixes {
		if strings.HasPrefix(raw, prefix) {
			return true
		}
	}
	return false
}
