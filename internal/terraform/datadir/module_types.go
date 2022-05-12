package datadir

import (
	"strings"

	tfaddr "github.com/hashicorp/terraform-registry-address"
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
	// Example: terraform-aws-modules/ec2-instance/aws
	// Example: registry.terraform.io/terraform-aws-modules/vpc/aws
	moduleSourceRegistry, err := tfaddr.ParseModuleSource(r.SourceAddr)
	if err == nil && moduleSourceRegistry.Package.Host == "registry.terraform.io" {
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
