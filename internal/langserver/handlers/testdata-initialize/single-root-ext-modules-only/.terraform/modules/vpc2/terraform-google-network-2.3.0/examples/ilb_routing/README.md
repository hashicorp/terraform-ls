# ILB routing example

This example configures a single VPC inside of a project.

This VPC has three subnets and a forwarding rule. Please note, that this is simply example resource usage, this module
wouldn't work as is.

More information:
- https://cloud.google.com/load-balancing/docs/internal/setting-up-ilb-next-hop
- https://cloud.google.com/load-balancing/docs/l7-internal/proxy-only-subnets

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| network\_name | The name of the VPC network being created | string | n/a | yes |
| project\_id | The project ID to host the network in | string | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| forwarding\_rule | Forwarding rule link |
| network\_name | The name of the VPC being created |
| network\_self\_link | The URI of the VPC being created |
| project\_id | VPC project id |
| route\_names | The routes associated with this VPC |
| subnets\_ips | The IP and cidrs of the subnets being created |
| subnets\_names | The names of the subnets being created |
| subnets\_regions | The region where subnets will be created |

<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
