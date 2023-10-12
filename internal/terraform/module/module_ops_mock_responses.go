// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package module

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	tfregistry "github.com/hashicorp/terraform-schema/registry"
	"github.com/zclconf/go-cty/cty"
)

// puppetModuleVersionsMockResponse represents response from https://registry.terraform.io/v1/modules/puppetlabs/deployment/ec/versions
var puppetModuleVersionsMockResponse = `{
  "modules": [
    {
      "source": "puppetlabs/deployment/ec",
      "versions": [
        {
          "version": "0.0.5",
          "root": {
            "providers": [
              {
                "name": "ec",
                "namespace": "",
                "source": "elastic/ec",
                "version": "0.2.1"
              }
            ],
            "dependencies": []
          },
          "submodules": []
        },
        {
          "version": "0.0.6",
          "root": {
            "providers": [
              {
                "name": "ec",
                "namespace": "",
                "source": "elastic/ec",
                "version": "0.2.1"
              }
            ],
            "dependencies": []
          },
          "submodules": []
        },
        {
          "version": "0.0.8",
          "root": {
            "providers": [
              {
                "name": "ec",
                "namespace": "",
                "source": "elastic/ec",
                "version": "0.2.1"
              }
            ],
            "dependencies": []
          },
          "submodules": []
        },
        {
          "version": "0.0.2",
          "root": {
            "providers": [
              {
                "name": "ec",
                "namespace": "",
                "source": "elastic/ec",
                "version": "0.2.1"
              }
            ],
            "dependencies": []
          },
          "submodules": []
        },
        {
          "version": "0.0.1",
          "root": {
            "providers": [],
            "dependencies": []
          },
          "submodules": [
            {
              "path": "modules/ec-deployment",
              "providers": [
                {
                  "name": "ec",
                  "namespace": "",
                  "source": "elastic/ec",
                  "version": "0.2.1"
                }
              ],
              "dependencies": []
            }
          ]
        },
        {
          "version": "0.0.4",
          "root": {
            "providers": [
              {
                "name": "ec",
                "namespace": "",
                "source": "elastic/ec",
                "version": "0.2.1"
              }
            ],
            "dependencies": []
          },
          "submodules": []
        },
        {
          "version": "0.0.3",
          "root": {
            "providers": [
              {
                "name": "ec",
                "namespace": "",
                "source": "elastic/ec",
                "version": "0.2.1"
              }
            ],
            "dependencies": []
          },
          "submodules": []
        },
        {
          "version": "0.0.7",
          "root": {
            "providers": [
              {
                "name": "ec",
                "namespace": "",
                "source": "elastic/ec",
                "version": "0.2.1"
              }
            ],
            "dependencies": []
          },
          "submodules": []
        }
      ]
    }
  ]
}`

// puppetModuleDataMockResponse represents response from https://registry.terraform.io/v1/modules/puppetlabs/deployment/ec/0.0.8
var puppetModuleDataMockResponse = `{
  "id": "puppetlabs/deployment/ec/0.0.8",
  "owner": "mattkirby",
  "namespace": "puppetlabs",
  "name": "deployment",
  "version": "0.0.8",
  "provider": "ec",
  "provider_logo_url": "/images/providers/generic.svg?2",
  "description": "",
  "source": "https://github.com/puppetlabs/terraform-ec-deployment",
  "tag": "v0.0.8",
  "published_at": "2021-08-05T00:26:33.501756Z",
  "downloads": 3059237,
  "verified": false,
  "root": {
    "path": "",
    "name": "deployment",
    "readme": "# EC project Terraform module\n\nTerraform module which creates a Elastic Cloud project.\n\n## Usage\n\nDetails coming soon\n",
    "empty": false,
    "inputs": [
      {
        "name": "autoscale",
        "type": "string",
        "description": "Enable autoscaling of elasticsearch",
        "default": "\"true\"",
        "required": false
      },
      {
        "name": "ec_stack_version",
        "type": "string",
        "description": "Version of Elastic Cloud stack to deploy",
        "default": "\"\"",
        "required": false
      },
      {
        "name": "name",
        "type": "string",
        "description": "Name of resources",
        "default": "\"ecproject\"",
        "required": false
      },
      {
        "name": "traffic_filter_sourceip",
        "type": "string",
        "description": "traffic filter source IP",
        "default": "\"\"",
        "required": false
      },
      {
        "name": "ec_region",
        "type": "string",
        "description": "cloud provider region",
        "default": "\"gcp-us-west1\"",
        "required": false
      },
      {
        "name": "deployment_templateid",
        "type": "string",
        "description": "ID of Elastic Cloud deployment type",
        "default": "\"gcp-io-optimized\"",
        "required": false
      }
    ],
    "outputs": [
      {
        "name": "elasticsearch_password",
        "description": "elasticsearch password"
      },
      {
        "name": "deployment_id",
        "description": "Elastic Cloud deployment ID"
      },
      {
        "name": "elasticsearch_version",
        "description": "Stack version deployed"
      },
      {
        "name": "elasticsearch_cloud_id",
        "description": "Elastic Cloud project deployment ID"
      },
      {
        "name": "elasticsearch_https_endpoint",
        "description": "elasticsearch https endpoint"
      },
      {
        "name": "elasticsearch_username",
        "description": "elasticsearch username"
      }
    ],
    "dependencies": [],
    "provider_dependencies": [
      {
        "name": "ec",
        "namespace": "elastic",
        "source": "elastic/ec",
        "version": "0.2.1"
      }
    ],
    "resources": [
      {
        "name": "ecproject",
        "type": "ec_deployment"
      },
      {
        "name": "gcp_vpc_nat",
        "type": "ec_deployment_traffic_filter"
      },
      {
        "name": "ec_tf_association",
        "type": "ec_deployment_traffic_filter_association"
      }
    ]
  },
  "submodules": [],
  "examples": [],
  "providers": [
    "ec"
  ],
  "versions": [
    "0.0.1",
    "0.0.2",
    "0.0.3",
    "0.0.4",
    "0.0.5",
    "0.0.6",
    "0.0.7",
    "0.0.8"
  ]
}`

// labelNullModuleVersionsMockResponse represents response for
// versions of module that suffers from "unreliable" input data, as described in
// https://github.com/hashicorp/vscode-terraform/issues/1582
// It is a shortened response from https://registry.terraform.io/v1/modules/cloudposse/label/null/versions
var labelNullModuleVersionsMockResponse = `{
  "modules": [
    {
      "source": "cloudposse/label/null",
      "versions": [
        {
          "version": "0.25.0",
          "root": {
            "providers": [],
            "dependencies": []
          },
          "submodules": []
        },
        {
          "version": "0.26.0",
          "root": {
            "providers": [],
            "dependencies": []
          },
          "submodules": []
        }
      ]
    }
  ]
}`

// labelNullModuleDataOldMockResponse represents response for
// a module that suffers from "unreliable" input data, as described in
// https://github.com/hashicorp/vscode-terraform/issues/1582
// It is a shortened response from https://registry.terraform.io/v1/modules/cloudposse/label/null/0.25.0
var labelNullModuleDataOldMockResponse = `{
  "id": "cloudposse/label/null/0.25.0",
  "owner": "osterman",
  "namespace": "cloudposse",
  "name": "label",
  "version": "0.25.0",
  "provider": "null",
  "provider_logo_url": "/images/providers/generic.svg?2",
  "description": "Terraform Module to define a consistent naming convention by (namespace, stage, name, [attributes])",
  "source": "https://github.com/cloudposse/terraform-null-label",
  "tag": "0.25.0",
  "published_at": "2021-08-25T17:47:04.039843Z",
  "downloads": 52863192,
  "verified": false,
  "root": {
    "path": "",
    "name": "label",
    "empty": false,
    "inputs": [
      {
        "name": "environment",
        "type": "string",
        "default": "",
        "required": true
      },
      {
        "name": "label_order",
        "type": "list(string)",
        "default": "",
        "required": true
      },
      {
        "name": "descriptor_formats",
        "type": "any",
        "default": "{}",
        "required": false
      }
    ],
    "outputs": [
      {
        "name": "id"
      }
    ],
    "dependencies": [],
    "provider_dependencies": [],
    "resources": []
  },
  "submodules": [],
  "examples": [],
  "providers": [
    "null",
    "terraform"
  ],
  "versions": [
    "0.25.0",
    "0.26.0"
  ]
}`

// labelNullModuleDataOldMockResponse represents response for
// a module that does NOT suffer from "unreliable" input data,
// as described in https://github.com/hashicorp/vscode-terraform/issues/1582
// This is for comparison with the unreliable input data.
var labelNullModuleDataNewMockResponse = `{
  "id": "cloudposse/label/null/0.26.0",
  "owner": "osterman",
  "namespace": "cloudposse",
  "name": "label",
  "version": "0.26.0",
  "provider": "null",
  "provider_logo_url": "/images/providers/generic.svg?2",
  "description": "Terraform Module to define a consistent naming convention by (namespace, stage, name, [attributes])",
  "source": "https://github.com/cloudposse/terraform-null-label",
  "tag": "0.26.0",
  "published_at": "2023-10-11T10:47:04.039843Z",
  "downloads": 10000,
  "verified": false,
  "root": {
    "path": "",
    "name": "label",
    "empty": false,
    "inputs": [
      {
        "name": "environment",
        "type": "string",
        "default": "",
        "required": true
      },
      {
        "name": "label_order",
        "type": "list(string)",
        "default": "null",
        "required": false
      },
      {
        "name": "descriptor_formats",
        "type": "any",
        "default": "{}",
        "required": false
      }
    ],
    "outputs": [
      {
        "name": "id"
      }
    ],
    "dependencies": [],
    "provider_dependencies": [],
    "resources": []
  },
  "submodules": [],
  "examples": [],
  "providers": [
    "null",
    "terraform"
  ],
  "versions": [
    "0.25.0",
    "0.26.0"
  ]
}`

var labelNullExpectedOldModuleData = &tfregistry.ModuleData{
	Version: version.Must(version.NewVersion("0.25.0")),
	Inputs: []tfregistry.Input{
		{
			Name:        "environment",
			Type:        cty.String,
			Description: lang.Markdown(""),
		},
		{
			Name:        "label_order",
			Type:        cty.DynamicPseudoType,
			Description: lang.Markdown(""),
		},
		{
			Name:        "descriptor_formats",
			Type:        cty.DynamicPseudoType,
			Description: lang.Markdown(""),
		},
	},
	Outputs: []tfregistry.Output{
		{
			Name:        "id",
			Description: lang.Markdown(""),
		},
	},
}

var labelNullExpectedNewModuleData = &tfregistry.ModuleData{
	Version: version.Must(version.NewVersion("0.26.0")),
	Inputs: []tfregistry.Input{
		{
			Name:        "environment",
			Type:        cty.String,
			Description: lang.Markdown(""),
			Required:    true,
		},
		{
			Name:        "label_order",
			Type:        cty.DynamicPseudoType,
			Description: lang.Markdown(""),
			Default:     cty.NullVal(cty.DynamicPseudoType),
		},
		{
			Name:        "descriptor_formats",
			Type:        cty.DynamicPseudoType,
			Description: lang.Markdown(""),
		},
	},
	Outputs: []tfregistry.Output{
		{
			Name:        "id",
			Description: lang.Markdown(""),
		},
	},
}
