// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/policy/state"
	"github.com/zclconf/go-cty/cty"
)

var resourcePolicyScopeId = lang.ScopeId("resource_policy")
var providerPolicyScopeId = lang.ScopeId("provider_policy")
var modulePolicyScopeId = lang.ScopeId("module_policy")

func builtinReferences(record *state.PolicyRecord) reference.Targets {
	targets := make(reference.Targets, 0)

	if record == nil {
		return targets
	}

	for _, rule := range record.Meta.ResourcePolicies {
		rng := rule.Range
		targets = append(targets, referenceResourcePolicyStaticTargets(rng)...)
	}

	for _, rule := range record.Meta.ProviderPolicies {
		rng := rule.Range
		targets = append(targets, referenceProviderPolicyStaticTargets(rng)...)
	}

	for _, rule := range record.Meta.ModulePolicies {
		rng := rule.Range
		targets = append(targets, referenceModulePolicyStaticTargets(rng)...)
	}

	return targets
}

func referenceResourcePolicyStaticTargets(rng hcl.Range) reference.Targets {
	return reference.Targets{
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "attrs"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                resourcePolicyScopeId,
			Type:                   cty.DynamicPseudoType,
			Description:            lang.Markdown("Attributes of the resource as defined by the provider"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "resource_type"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                resourcePolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("Resource type (specified in the label when wildcard is not used)"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "provider_type"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                resourcePolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("Provider powering the resource"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "tfe_workspace"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                resourcePolicyScopeId,
			Type:                   cty.DynamicPseudoType,
			Description:            lang.Markdown("Workspace config for which the resource belongs to"),
		},
	}
}

func referenceProviderPolicyStaticTargets(rng hcl.Range) reference.Targets {
	return reference.Targets{
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "attrs"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.DynamicPseudoType,
			Description:            lang.Markdown("Attributes of the resource as defined by the provider"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "name"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("Local identifier for the provider"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "alias"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("Local alias of the provider"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "type"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The official, short name of the provider. This is the simple identifier used to declare a provider block or resource type"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "namespace"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("In the context of the registry, this is the organization or user who publishes the provider. It is the first segment of the provider's source address"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "source"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The full, canonical registry address used to locate and download the provider plugin. It combines the namespace and the type"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "module_path"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("Root module path"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "version"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("Version of the provider"),
		},
	}
}

func referenceModulePolicyStaticTargets(rng hcl.Range) reference.Targets {
	return reference.Targets{
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "attrs"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                modulePolicyScopeId,
			Type:                   cty.DynamicPseudoType,
			Description:            lang.Markdown("Attributes of the resource as defined by the provider"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "address"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                modulePolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The `address` is the internal, logical path used by Terraform to reference resources within a configuration for commands like state management"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "source"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                modulePolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The `source` is the external location where Terraform physically finds and downloads the module code"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "version"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                modulePolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("Version of the module"),
		},
	}
}
