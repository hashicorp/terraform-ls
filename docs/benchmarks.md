# Benchmarks

There is a few factors which affect how much CPU and memory the language server uses.

 - amount of (initialized or referenced) modules within any workspace
 - amount of providers used within these modules
 - size of the providers (schema) used within these modules (i.e. amount of resources, data sources and attributes within)

While we generally aim to keep both CPU and memory usage low, we'd consider optimizing for lower CPU usage as a higher priority than lower memory usage. i.e. we trade lower CPU for higher memory where such a trade-off is necessary.

We optimize for what we consider the common case, which is approximately 1-100 reasonably sized modules, each typically with a single provider, within a workspace.

## Benchmarked Modules

We run benchmarks with the following modules which estimates the **average time to index** + **average memory allocation** on `Standard_DS2_v2` MS Azure VMs (via [GitHub-hosted runners](https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners#cloud-hosts-for-github-hosted-runners)).

 - [nearly empty local module with no submodule and no provider](../internal/langserver/handlers/testdata/single-module-no-provider)
   - `35ms`
   - `1.4MB`
 - [local module with a single submodule, no provider](../internal/langserver/handlers/testdata/single-submodule-no-provider)
   - `225ms`
   - `2.6MB`
 - [local module with the `random` provider](../internal/langserver/handlers/testdata/single-module-random)
   - `220ms`
   - `1.9MB`
 - [local module with the `aws` provider](../internal/langserver/handlers/testdata/single-module-aws)
   - `1.5s`
   - `102MB`
 - [aws-consul](https://github.com/hashicorp/terraform-aws-consul)
   - `1.6s`
   - `123MB`
 - [aws-eks](https://registry.terraform.io/modules/terraform-aws-modules/eks/aws)
   - `1.9s`
   - `137MB`
 - [aws-vpc](https://registry.terraform.io/modules/terraform-aws-modules/vpc/aws)
   - `1.7s`
   - `120MB`
 - [google-project](https://registry.terraform.io/modules/terraform-google-modules/project-factory/google)
   - `1.9s`
   - `145MB`
 - [google-network](https://registry.terraform.io/modules/terraform-google-modules/network/google)
   - `1.8s`
   - `130MB`
 - [google-gke](https://registry.terraform.io/modules/terraform-google-modules/kubernetes-engine/google)
   - `3.3s`
   - `129MB`
 - [k8s-metrics-server](https://registry.terraform.io/modules/cookielab/metrics-server/kubernetes)
   - `1.6s`
   - `59MB`
 - [k8s-dashboard](https://registry.terraform.io/modules/cookielab/dashboard/kubernetes)
   - `2.0s`
   - `59MB`

Sections below provide more details on the factors affecting the usage and usage patterns.

## CPU usage

The server is expected to consume a little more CPU upon launch until it indexes the workspace and runs various commands in these folders, such as `terraform providers schema -json` or `terraform version`.

Based on benchmarking various publicly available modules, we expect indexing of a single module to take between `200ms` and `2s`. This will vary depending on the amount of installed _submodules_ of that module and size of provider schema (with AWS representing possibly the largest publicly known provider).

The indexers currently don't attempt to utilize more than 1 CPU core (i.e. parallelise), to reduce CPU spikes but we may consider making this opt-in in the future to allow users trade reduced indexing time for higher CPU usage where this makes sense.

## Memory usage

The server is expected to consume around 300 MB upon launch without any open/indexed files. The majority of these 300 MB is consumed by the embedded schemas of approximately 200 providers.

Every open file will be stored in memory along with its AST, diagnostics, references and other metadata.
Similar to embedded schemas, any schemas obtained locally from any installed providers via `terraform providers schema -json` will be persisted and likely take up most of the total memory footprint of the server.
