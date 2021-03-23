# Q1 2021 Roadmap

Each quarter, the team will highlight areas of focus for our work and upcoming research.
Each release will include necessary tasks that lead to the completion of the stated goals as well as community pull requests, enhancements, and features that are not highlighted in the roadmap. This calendar quarter (Jan-Mar 2021) we will be prioritizing the following areas of work:

## Currently In Progress
### Expanded Completion and Hover
The `terraform-ls` language server supports basic schema-driven completion. We plan to introduce additional completion and hover capabilities:

- Provide nested navigation symbols (i.e. nested blocks and block attributes) [#420](https://github.com/hashicorp/terraform-ls/pull/420) :white_check_mark:
- Modules

### Syntax highlighting improvements via semantic tokens [#331](https://github.com/hashicorp/terraform-ls/pull/331) :white_check_mark:

### Provide completion and hover for expressions (i.e. references such as `aws_instance.public_ip`) [#38](https://github.com/hashicorp/terraform-ls/issues/38)

### Improved detection of uninitialized modules

## Researching
- Improve HCL identification and interpretation within LSP
- Add support for sourcing dynamic provider schemas

## Disclosures
The product-development initiatives in this document reflect HashiCorp's current plans and are subject to change and/or cancellation at HashiCorp's sole discretion.
