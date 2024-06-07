# Upgrading to v2.x

The v2.x release of _google-network_ is a backwards incompatible
release.

Because v2.x changed how the subnet resource is iterated on, resources in Terraform state need to be migrated in order to avoid the resources from getting destroyed and recreated.

## Output Changes
In version 2.x, a few output names were [changed](https://github.com/terraform-google-modules/terraform-google-network/compare/v1.5.0...v2.0.0#diff-c09d00f135e3672d079ff6e0556d957d):

- `svpc_host_project_id` was renamed to `project_id`.
- `routes` was renamed to `route_names`

## Migration Instructions

First, upgrade to the new version of this module.

```diff
 module "kubernetes_engine_private_cluster" {
   source  = "terraform-google-modules/network/google"
-  version = "~> 1.5"
+  version = "~> 2.0"

   # ...
 }
```

If you run `terraform plan` at this point, Terraform will inform you that it will attempt to delete and recreate your existing subnets. This is almost certainly not the behavior you want.

You will need to migrate your state, either [manually](#manual-migration-steps) or [automatically](#migration-script).

### Migration Script

1.  Download the script:

    ```sh
    curl -O https://raw.githubusercontent.com/terraform-google-modules/terraform-google-network/master/helpers/migrate.py
    chmod +x migrate.py
    ```

2.  Back up your Terraform state:

    ```sh
    terraform state pull >> state.bak
    ```

2.  Run the script to output the migration commands:

    ```sh
    $  ./migrate.py --dryrun
    terraform state mv 'module.example.module.test-vpc-module-02.google_compute_network.network[0]' 'module.example.module.test-vpc-module-02.module.vpc.google_compute_network.network'
    terraform state mv 'module.example.module.test-vpc-module-02.google_compute_subnetwork.subnetwork' 'module.example.module.test-vpc-module-02.module.subnets.google_compute_subnetwork.subnetwork'
    terraform state mv 'module.example.module.test-vpc-module-02.module.subnets.google_compute_subnetwork.subnetwork[0]' 'module.example.module.test-vpc-module-02.module.subnets.google_compute_subnetwork.subnetwork["us-west1/multi-vpc-a1-02-subnet-01"]'
    terraform state mv 'module.example.module.test-vpc-module-02.module.subnets.google_compute_subnetwork.subnetwork[1]' 'module.example.module.test-vpc-module-02.module.subnets.google_compute_subnetwork.subnetwork["us-west1/multi-vpc-a1-02-subnet-02"]'
    terraform state mv 'module.example.module.test-vpc-module-02.google_compute_route.route' 'module.example.module.test-vpc-module-02.module.routes.google_compute_route.route'
    terraform state mv 'module.example.module.test-vpc-module-02.module.routes.google_compute_route.route[0]' 'module.example.module.test-vpc-module-02.module.routes.google_compute_route.route["multi-vpc-a1-02-egress-inet"]'
    terraform state mv 'module.example.module.test-vpc-module-02.module.routes.google_compute_route.route[1]' 'module.example.module.test-vpc-module-02.module.routes.google_compute_route.route["multi-vpc-a1-02-testapp-proxy"]'

    ```

3.  Execute the migration script:

    ```sh
    $ ./migrate.py
    ---- Migrating the following modules:
    -- module.example.module.test-vpc-module-02
    ---- Commands to run:
    Move "module.example.module.test-vpc-module-02.google_compute_network.network[0]" to "module.example.module.test-vpc-module-02.module.vpc.google_compute_network.network"
    Successfully moved 1 object(s).
    Move "module.example.module.test-vpc-module-02.google_compute_subnetwork.subnetwork" to "module.example.module.test-vpc-module-02.module.subnets.google_compute_subnetwork.subnetwork"
    Successfully moved 1 object(s).
    Move "module.example.module.test-vpc-module-02.module.subnets.google_compute_subnetwork.subnetwork[0]" to "module.example.module.test-vpc-module-02.module.subnets.google_compute_subnetwork.subnetwork[\"us-west1/multi-vpc-a1-02-subnet-01\"]"
    Successfully moved 1 object(s).
    Move "module.example.module.test-vpc-module-02.module.subnets.google_compute_subnetwork.subnetwork[1]" to "module.example.module.test-vpc-module-02.module.subnets.google_compute_subnetwork.subnetwork[\"us-west1/multi-vpc-a1-02-subnet-02\"]"
    Successfully moved 1 object(s).
    Move "module.example.module.test-vpc-module-02.google_compute_route.route" to "module.example.module.test-vpc-module-02.module.routes.google_compute_route.route"
    Successfully moved 1 object(s).
    Move "module.example.module.test-vpc-module-02.module.routes.google_compute_route.route[0]" to "module.example.module.test-vpc-module-02.module.routes.google_compute_route.route[\"multi-vpc-a1-02-egress-inet\"]"
    Successfully moved 1 object(s).
    Move "module.example.module.test-vpc-module-02.module.routes.google_compute_route.route[1]" to "module.example.module.test-vpc-module-02.module.routes.google_compute_route.route[\"multi-vpc-a1-02-testapp-proxy\"]"
    Successfully moved 1 object(s).

    ```

4.  Run `terraform plan` to confirm no changes are expected.

### Manual Migration Steps

In this example here are the commands used migrate the vpc and subnets created by the `simple_project` in the examples directory.  _please note the need to escape the quotes on the new resource_. You may also use the migration script.

-   `terraform state mv module.example.module.test-vpc-module.google_compute_network.network module.example.module.test-vpc-module.module.vpc.google_compute_subnetwork.network`

-   `terraform state mv module.example.module.test-vpc-module.google_compute_subnetwork.subnetwork module.example.module.test-vpc-module.module.subnets.google_compute_subnetwork.subnetwork`

-   `terraform state mv module.example.module.test-vpc-module.module.subnets.google_compute_subnetwork.subnetwork[0] module.example.module.test-vpc-module.module.subnets.google_compute_subnetwork.subnetwork[\"us-west1/simple-project-timh-subnet-01\"]`

-   `terraform state mv module.example.module.test-vpc-module.module.subnets.google_compute_subnetwork.subnetwork[1] module.example.module.test-vpc-module.module.subnets.google_compute_subnetwork.subnetwork[\"us-west1/simple-project-timh-subnet-02\"]`

*You'll notice that because of a terraform [issue](https://github.com/hashicorp/terraform/issues/22301), we need to move the whole resource collection first before renaming to the `for_each` keys*

`terraform plan` should now return a no-op and show no new changes.

```Shell
$ terraform plan
Refreshing Terraform state in-memory prior to plan...
The refreshed state will be used to calculate this plan, but will not be
persisted to local or remote state storage.

module.example.module.test-vpc-module.google_compute_network.network: Refreshing state... [id=simple-project-timh]
module.example.module.test-vpc-module.google_compute_subnetwork.subnetwork["us-west1/simple-project-timh-subnet-02"]: Refreshing state... [id=us-west1/simple-project-timh-subnet-02]
module.example.module.test-vpc-module.google_compute_subnetwork.subnetwork["us-west1/simple-project-timh-subnet-01"]: Refreshing state... [id=us-west1/simple-project-timh-subnet-01]

------------------------------------------------------------------------

No changes. Infrastructure is up-to-date.

This means that Terraform did not detect any differences between your
configuration and real physical resources that exist. As a result, no
actions need to be performed.
```

### Known Issues

If your previous state only contains a **single** subnet or route then `terraform mv` will throw an error similar to the following during migration:

```
Error: Invalid target address

Cannot move to
module.example.module.test-vpc-module-01.module.routes.google_compute_route.route["multi-vpc-a1-01-egress-inet"]:
module.example.module.test-vpc-module-01.module.routes.google_compute_route.route
does not exist in the current state.
```

This is due to a terraform mv [issue](https://github.com/hashicorp/terraform/issues/22301)

The workaround is to either

1. Create a temporary subnet or route prior to migration
2. Manually updating the state file. Update the `index_key` of the appropriate user and push the to the remote state if necessary.
