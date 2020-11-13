# Q4 2020 Roadmap

Each quarter, the team will highlight areas of focus for our work and upcoming research.
Each release will include necessary tasks that lead to the completion of the stated goals as well as community pull requests, enhancements, and features that are not highlighted in the roadmap. This calendar quarter (Oct-Dec 2020) we will be prioritizing the following areas of work:

## Currently In Progress
### Expanded Completion and Hover
The `terraform-ls` language server supports basic schema-driven completion. We plan to introduce additional completion and hover capabilities:

- Modules
- Provide nested navigation symbols (i.e. nested blocks and block attributes)

### Documentation on hover [#294](https://github.com/hashicorp/terraform-ls/pull/294)

### Built-in block types (ie. Terraform block, backends, provisioners, variables, locals, outputs) [#287](https://github.com/hashicorp/terraform-ls/pull/287)

### Support module wide diagnostics [#288](https://github.com/hashicorp/terraform-ls/pull/288)

### Support for upcoming Terraform v0.14 [#289](https://github.com/hashicorp/terraform-ls/pull/288)

### Dedicated HCL decoder [#281](https://github.com/hashicorp/terraform-ls/pull/281)

### Progressively enhanced completion as more core or provider schema is discovered [#281](https://github.com/hashicorp/terraform-ls/pull/281)

## Researching
- Improve HCL identification and interpretation within LSP
- Provide completion and hover for expressions (i.e. references such as `aws_instance.public_ip`)
- Investigate integration of other diagnostic helpers, such as [tflint](https://github.com/terraform-linters/tflint)

## Disclosures
The product-development initiatives in this document reflect HashiCorp's current plans and are subject to change and/or cancellation at HashiCorp's sole discretion.
