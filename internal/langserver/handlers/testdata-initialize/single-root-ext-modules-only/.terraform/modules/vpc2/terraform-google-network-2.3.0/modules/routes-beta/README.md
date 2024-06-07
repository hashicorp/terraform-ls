# Terraform Network Beta Module

This submodule is part of the the `terraform-google-network` module. It creates the individual vpc routes and optionally deletes the default internet gateway routes.

It supports creating:

- Routes within vpc network.
- Optionally deletes the default internet gateway routes.

It also uses google beta provider to support the following resource fields:

-  google_compute_route.next_hop_ilb

## Usage

Basic usage of this submodule is as follows:

```hcl
module "vpc" {
    source  = "terraform-google-modules/network/google//modules/routes-beta"
    version = "~> 2.0.0"

    project_id   = "<PROJECT ID>"
    network_name = "example-vpc"

    delete_default_internet_gateway_routes = false

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
        {
            name                   = "test-proxy"
            description            = "route through idp to reach app"
            destination_range      = "10.50.10.0/24"
            tags                   = "app-proxy"
            next_hop_ilb           = var.ilb_link
        },
    ]
}
```

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| delete\_default\_internet\_gateway\_routes | If set, ensure that all routes within the network specified whose names begin with 'default-route' and with a next hop of 'default-internet-gateway' are deleted | string | `"false"` | no |
| module\_depends\_on | List of modules or resources this module depends on. | list | `<list>` | no |
| network\_name | The name of the network where routes will be created | string | n/a | yes |
| project\_id | The ID of the project where the routes will be created | string | n/a | yes |
| routes | List of routes being created in this VPC | list(map(string)) | `<list>` | no |
| routes\_count | Amount of routes being created in this VPC | number | `"0"` | no |

## Outputs

| Name | Description |
|------|-------------|
| routes | The created routes resources |

<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->


### Routes Input

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
