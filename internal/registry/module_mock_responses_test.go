// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package registry

// moduleVersionsMockResponse represents response from https://registry.terraform.io/v1/modules/puppetlabs/deployment/ec/versions
var moduleVersionsMockResponse = `{
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

// moduleDataMockResponse represents response from https://registry.terraform.io/v1/modules/puppetlabs/deployment/ec/0.0.8
var moduleDataMockResponse = `{
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
