# Terraform Network Module

This modules makes it easy to set up a new VPC Network in GCP by defining your network and subnet ranges in a concise syntax.

It supports creating:

- A Google Virtual Private Network (VPC)
- Subnets within the VPC
- Secondary ranges for the subnets (if applicable)

Sub modules are provided for creating individual vpc, subnets, and routes. See the modules directory for the various sub modules usage.

## Compatibility

This module is meant for use with Terraform 0.12. If you haven't [upgraded](https://www.terraform.io/upgrade-guides/0-12.html) and need a Terraform 0.11.x-compatible version of this module, the last released version intended for Terraform 0.11.x is [0.8.0](https://registry.terraform.io/modules/terraform-google-modules/network/google/0.8.0).

## Usage
You can go to the examples folder, however the usage of the module could be like this in your own main.tf file:

```hcl
module "vpc" {
    source  = "terraform-google-modules/network/google"
    version = "~> 2.3"

    project_id   = "<PROJECT ID>"
    network_name = "example-vpc"
    routing_mode = "GLOBAL"

    subnets = [
        {
            subnet_name           = "subnet-01"
            subnet_ip             = "10.10.10.0/24"
            subnet_region         = "us-west1"
        },
        {
            subnet_name           = "subnet-02"
            subnet_ip             = "10.10.20.0/24"
            subnet_region         = "us-west1"
            subnet_private_access = "true"
            subnet_flow_logs      = "true"
            description           = "This subnet has a description"
        },
        {
            subnet_name               = "subnet-03"
            subnet_ip                 = "10.10.30.0/24"
            subnet_region             = "us-west1"
            subnet_flow_logs          = "true"
            subnet_flow_logs_interval = "INTERVAL_10_MIN"
            subnet_flow_logs_sampling = 0.7
            subnet_flow_logs_metadata = "INCLUDE_ALL_METADATA"
        }
    ]

    secondary_ranges = {
        subnet-01 = [
            {
                range_name    = "subnet-01-secondary-01"
                ip_cidr_range = "192.168.64.0/24"
            },
        ]

        subnet-02 = []
    }

    routes = [
        {
            name                   = "egress-internet"
            description            = "route through IGW to access internet"
            destination_range      = "0.0.0.0/0"
            tags                   = "egress-inet"
            next_hop_internet      = "true"
        },
        {
            name                   = "app-proxy"
            description            = "route through proxy to reach app"
            destination_range      = "10.50.10.0/24"
            tags                   = "app-proxy"
            next_hop_instance      = "app-proxy-instance"
            next_hop_instance_zone = "us-west1-a"
        },
    ]
}
```

Then perform the following commands on the root folder:

- `terraform init` to get the plugins
- `terraform plan` to see the infrastructure plan
- `terraform apply` to apply the infrastructure build
- `terraform destroy` to destroy the built infrastructure

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| auto\_create\_subnetworks | When set to true, the network is created in 'auto subnet mode' and it will create a subnet for each region automatically across the 10.128.0.0/9 address range. When set to false, the network is created in 'custom subnet mode' so the user can explicitly connect subnetwork resources. | bool | `"false"` | no |
| delete\_default\_internet\_gateway\_routes | If set, ensure that all routes within the network specified whose names begin with 'default-route' and with a next hop of 'default-internet-gateway' are deleted | string | `"false"` | no |
| description | An optional description of this resource. The resource must be recreated to modify this field. | string | `""` | no |
| network\_name | The name of the network being created | string | n/a | yes |
| project\_id | The ID of the project where this VPC will be created | string | n/a | yes |
| routes | List of routes being created in this VPC | list(map(string)) | `<list>` | no |
| routing\_mode | The network routing mode (default 'GLOBAL') | string | `"GLOBAL"` | no |
| secondary\_ranges | Secondary ranges that will be used in some of the subnets | object | `<map>` | no |
| shared\_vpc\_host | Makes this project a Shared VPC host if 'true' (default 'false') | bool | `"false"` | no |
| subnets | The list of subnets being created | list(map(string)) | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| network | The created network |
| network\_name | The name of the VPC being created |
| network\_self\_link | The URI of the VPC being created |
| project\_id | VPC project id |
| route\_names | The route names associated with this VPC |
| subnets | A map with keys of form subnet_region/subnet_name and values being the outputs of the google_compute_subnetwork resources used to create corresponding subnets. |
| subnets\_flow\_logs | Whether the subnets will have VPC flow logs enabled |
| subnets\_ips | The IPs and CIDRs of the subnets being created |
| subnets\_names | The names of the subnets being created |
| subnets\_private\_access | Whether the subnets will have access to Google API's without a public IP |
| subnets\_regions | The region where the subnets will be created |
| subnets\_secondary\_ranges | The secondary ranges associated with these subnets |
| subnets\_self\_links | The self-links of subnets being created |

<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->

### Subnet Inputs

The subnets list contains maps, where each object represents a subnet. Each map has the following inputs (please see examples folder for additional references):

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| subnet\_name | The name of the subnet being created  | string | - | yes |
| subnet\_ip | The IP and CIDR range of the subnet being created | string | - | yes |
| subnet\_region | The region where the subnet will be created  | string | - | yes |
| subnet\_private\_access | Whether this subnet will have private Google access enabled | string | `"false"` | no |
| subnet\_flow\_logs  | Whether the subnet will record and send flow log data to logging | string | `"false"` | no |

### Route Inputs

The routes list contains maps, where each object represents a route. For the next_hop_* inputs, only one is possible to be used in each route. Having two next_hop_* inputs will produce an error. Each map has the following inputs (please see examples folder for additional references):

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| name | The name of the route being created  | string | - | no |
| description | The description of the route being created | string | - | no |
| tags | The network tags assigned to this route. This is a list in string format. Eg. "tag-01,tag-02"| string | - | yes |
| destination\_range | The destination range of outgoing packets that this route applies to. Only IPv4 is supported | string | - | yes
| next\_hop\_internet | Whether the next hop to this route will the default internet gateway. Use "true" to enable this as next hop | string | `"false"` | yes |
| next\_hop\_ip | Network IP address of an instance that should handle matching packets | string | - | yes |
| next\_hop\_instance |  URL or name of an instance that should handle matching packets. If just name is specified "next\_hop\_instance\_zone" is required | string | - | yes |
| next\_hop\_instance\_zone |  The zone of the instance specified in next\_hop\_instance. Only required if next\_hop\_instance is specified as a name | string | - | no |
| next\_hop\_vpn\_tunnel | URL to a VpnTunnel that should handle matching packets | string | - | yes |
| priority | The priority of this route. Priority is used to break ties in cases where there is more than one matching route of equal prefix length. In the case of two routes with equal prefix length, the one with the lowest-numbered priority value wins | string | `"1000"` | yes |

## Requirements
### Installed Software
- [Terraform](https://www.terraform.io/downloads.html) ~> 0.12.6
- [Terraform Provider for GCP](https://github.com/terraform-providers/terraform-provider-google) ~> 2.19
- [Terraform Provider for GCP Beta](https://github.com/terraform-providers/terraform-provider-google-beta) ~>
  2.19
- [gcloud](https://cloud.google.com/sdk/gcloud/) >243.0.0

### Configure a Service Account
In order to execute this module you must have a Service Account with the following roles:

- roles/compute.networkAdmin on the organization or folder

If you are going to manage a Shared VPC, you must have either:

- roles/compute.xpnAdmin on the organization
- roles/compute.xpnAdmin on the folder (beta)

### Enable API's
In order to operate with the Service Account you must activate the following API on the project where the Service Account was created:

- Compute Engine API - compute.googleapis.com

## Contributing

Refer to the [contribution guidelines](./CONTRIBUTING.md) for
information on contributing to this module.
