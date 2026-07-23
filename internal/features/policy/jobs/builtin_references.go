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
				lang.RootStep{Name: "prior_attrs"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                resourcePolicyScopeId,
			Type:                   cty.DynamicPseudoType,
			Description:            lang.Markdown("Prior state of resource attributes before the change"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "module_path"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                resourcePolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The module path the resource belongs to"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "operation"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                resourcePolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The operation being performed (create / update / delete)"),
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
				lang.AttrStep{Name: "type"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                resourcePolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The resource type, this is the first label of the matching resource block"),
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
				lang.AttrStep{Name: "source"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The full source of the provider"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "version"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The resolved version of the provider"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "alias"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("Alias given to the provider"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "name"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The local name of the provider"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "namespace"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                providerPolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The provider's registry namespace"),
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
			Description:            lang.Markdown("Provides access to the module's input variables. The available attributes depend on the input variables each Terraform module defines"),
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
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "meta"},
				lang.AttrStep{Name: "address"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                modulePolicyScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The logical `address` of the module within the configuration"),
		},
	}
}
