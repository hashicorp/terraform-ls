# Google Cloud Shared VPC Access Configuration

This module allows configuring service project access to a Shared VPC, created with the top-level network module. The module allows:

- attaching service projects to the Shared VPC host project
- assigning IAM roles for each Shared VPC subnet

Full details on service project configuration can be found in the Google Cloud documentation on *[Provisioning Shared VPC](https://cloud.google.com/vpc/docs/provisioning-shared-vpc)*, and to *[Setting up clusters with Shared VPC](https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc)*. Details and use cases of using service accounts as role recipients for Shared VPC are in the *[Service accounts as project admins](https://cloud.google.com/vpc/docs/provisioning-shared-vpc#sa-as-spa)* section of the first document above.

The resources created/managed by this module are:

- one `google_compute_shared_vpc_service_project` resource for each project where full VPC access is needed
- one `google_compute_subnetwork_iam_binding` for each subnetwork where individual subnetwork access is needed

## Usage

Basic usage of this module is as follows:

```hcl
module "net-shared-vpc-access" {
  source              = "terraform-google-modules/network/google//modules/fabric-net-svpc-access"
  version             = "~> 1.4.0"
  host_project_id     = "my-host-project-id"
  service_project_num = 1
  service_project_ids = ["my-service-project-id"]
  host_subnets        = ["my-subnet"]
  host_subnet_regions = ["europe-west1"]
  host_subnet_users   = {
    my-subnet = "group:my-service-owners@example.org,serviceAccount:1234567890@cloudservices.gserviceaccount.com"
  }
  host_service_agent_role = true
  host_service_agent_users = [
    "serviceAccount:service-123456789@container-engine-robot.iam.gserviceaccount.com"
  ]
}
```

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| host\_project\_id | Project id of the shared VPC host project. | string | n/a | yes |
| host\_service\_agent\_role | Assign host service agent role to users in host_service_agent_users variable. | bool | `"false"` | no |
| host\_service\_agent\_users | List of IAM-style users that will be granted the host service agent role on the host project. | list(string) | `<list>` | no |
| host\_subnet\_regions | List of subnet regions, one per subnet. | list(string) | `<list>` | no |
| host\_subnet\_users | Map of comma-delimited IAM-style members to which network user roles for subnets will be assigned. | map(any) | `<map>` | no |
| host\_subnets | List of subnet names on which to grant network user role. | list(string) | `<list>` | no |
| service\_project\_ids | Ids of the service projects that will be attached to the Shared VPC. | list(string) | n/a | yes |
| service\_project\_num | Number of service projects that will be attached to the Shared VPC. | number | `"0"` | no |

## Outputs

| Name | Description |
|------|-------------|
| service\_projects | Project ids of the services with access to all subnets. |

<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
