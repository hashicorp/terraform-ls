# Delete Default Gateway Routes

This example configures a single simple VPC inside of a project.

This VPC has a single subnet with no secondary ranges, and ensures the default internet gateway route is deleted.

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| network\_name | The name of the VPC network being created | string | n/a | yes |
| project\_id | The project ID to host the network in | string | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| network\_name | The name of the VPC being created |
| network\_self\_link | The URI of the VPC being created |
| route\_names | The routes associated with this VPC |
| subnets\_flow\_logs | Whether the subnets will have VPC flow logs enabled |
| subnets\_ips | The IP and cidrs of the subnets being created |
| subnets\_names | The names of the subnets being created |
| subnets\_private\_access | Whether the subnets will have access to Google API's without a public IP |
| subnets\_regions | The region where subnets will be created |
| subnets\_secondary\_ranges | The secondary ranges associated with these subnets |

<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
