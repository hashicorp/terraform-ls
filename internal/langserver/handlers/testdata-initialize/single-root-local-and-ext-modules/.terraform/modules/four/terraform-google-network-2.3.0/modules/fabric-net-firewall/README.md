# Google Cloud VPC Firewall

This module allows creation of a minimal VPC firewall, supporting basic configurable rules for IP range-based intra-VPC and administrator ingress,  tag-based SSH/HTTP/HTTPS ingress, and custom rule definitions.

The HTTP and HTTPS rules use the same network tags that are assigned to instances when the "Allow HTTP[S] traffic" checkbox is flagged in the Cloud Console. The SSH rule uses a generic `ssh` tag.

All IP source ranges are configurable through variables, and are set by default to `0.0.0.0/0` for tag-based rules. Allowed protocols and/or ports for the intra-VPC rule are also configurable through a variable.

Custom rules are set through a map where keys are rule names, and values use this custom type:

```hcl
map(object({
  description          = string
  direction            = string       # (INGRESS|EGRESS)
  action               = string       # (allow|deny)
  ranges               = list(string) # list of IP CIDR ranges
  sources              = list(string) # tags or SAs (ignored for EGRESS)
  targets              = list(string) # tags or SAs
  use_service_accounts = bool         # use tags or SAs in sources/targets
  rules = list(object({
    protocol = string
    ports    = list(string)
  }))
  extra_attributes = map(string)      # map, optional keys disabled or priority
}))
```

The resources created/managed by this module are:

- one optional ingress rule from internal CIDR ranges, only allowing ICMP by default
- one optional ingress rule from admin CIDR ranges, allowing all protocols on all ports
- one optional ingress rule for SSH on network tag `ssh`
- one optional ingress rule for HTTP on network tag `http-server`
- one optional ingress rule for HTTPS on network tag `https-server`
- one or more optional custom rules


## Usage

Basic usage of this module is as follows:

```hcl
module "net-firewall" {
  source                  = "terraform-google-modules/network/google//modules/fabric-net-firewall"
  project_id              = "my-project"
  network                 = "my-vpc"
  internal_ranges_enabled = true
  internal_ranges         = ["10.0.0.0/0"]
  custom_rules = {
    ingress-sample = {
      description          = "Dummy sample ingress rule, tag-based."
      direction            = "INGRESS"
      action               = "allow"
      ranges               = ["192.168.0.0"]
      sources              = ["spam-tag"]
      targets              = ["foo-tag", "egg-tag"]
      use_service_accounts = false
      rules = [
        {
          protocol = "tcp"
          ports    = []
        }
      ]
      extra_attributes = {}
    }
  }
}
```

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| admin\_ranges | IP CIDR ranges that have complete access to all subnets. | list | `<list>` | no |
| admin\_ranges\_enabled | Enable admin ranges-based rules. | string | `"false"` | no |
| custom\_rules | List of custom rule definitions (refer to variables file for syntax). | object | `<map>` | no |
| http\_source\_ranges | List of IP CIDR ranges for tag-based HTTP rule, defaults to 0.0.0.0/0. | list | `<list>` | no |
| https\_source\_ranges | List of IP CIDR ranges for tag-based HTTPS rule, defaults to 0.0.0.0/0. | list | `<list>` | no |
| internal\_allow | Allow rules for internal ranges. | list | `<list>` | no |
| internal\_ranges | IP CIDR ranges for intra-VPC rules. | list | `<list>` | no |
| internal\_ranges\_enabled | Create rules for intra-VPC ranges. | string | `"false"` | no |
| network | Name of the network this set of firewall rules applies to. | string | n/a | yes |
| project\_id | Project id of the project that holds the network. | string | n/a | yes |
| ssh\_source\_ranges | List of IP CIDR ranges for tag-based SSH rule, defaults to 0.0.0.0/0. | list | `<list>` | no |

## Outputs

| Name | Description |
|------|-------------|
| admin\_ranges | Admin ranges data. |
| custom\_egress\_allow\_rules | Custom egress rules with allow blocks. |
| custom\_egress\_deny\_rules | Custom egress rules with allow blocks. |
| custom\_ingress\_allow\_rules | Custom ingress rules with allow blocks. |
| custom\_ingress\_deny\_rules | Custom ingress rules with deny blocks. |
| internal\_ranges | Internal ranges. |

<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
