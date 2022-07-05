package datadir

import (
	"strings"

	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

type ModuleType string

const (
	UNKNOWN    ModuleType = "unknown"
	TFREGISTRY ModuleType = "tfregistry"
	LOCAL      ModuleType = "local"
	GITHUB     ModuleType = "github"
	GIT        ModuleType = "git"
)

// GetModuleType parses source addresses to determine what kind of source the Terraform module comes
// from. It currently supports detecting Terraform Registry modules, GitHub modules, Git modules, and
// local file paths
func GetModuleType(sourceAddr string) ModuleType {
	// Example: terraform-aws-modules/ec2-instance/aws
	// Example: registry.terraform.io/terraform-aws-modules/vpc/aws
	moduleSourceRegistry, err := tfaddr.ParseModuleSource(sourceAddr)
	if err == nil && moduleSourceRegistry.Package.Host == "registry.terraform.io" {
		return TFREGISTRY
	}

	// Example: github.com/terraform-aws-modules/terraform-aws-security-group
	if strings.HasPrefix(sourceAddr, "github.com/") {
		return GITHUB
	}

	// Example: git::https://example.com/vpc.git
	if strings.HasPrefix(sourceAddr, "git::") {
		return GIT
	}

	// Local file paths
	if isModuleSourceLocal(sourceAddr) {
		return LOCAL
	}

	return UNKNOWN
}

func isModuleSourceLocal(raw string) bool {
	for _, prefix := range tfmod.ModuleSourceLocalPrefixes {
		if strings.HasPrefix(raw, prefix) {
			return true
		}
	}
	return false
}
