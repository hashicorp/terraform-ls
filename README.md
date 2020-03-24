# Terraform Language Server

Experimental version of [Terraform](https://www.terraform.io) Language Server.

## What is LSP

Read more about the Language Server Protocol at https://microsoft.github.io/language-server-protocol/

## Current Status

Not all LSP or language features are available at the time of writing,
but this is an active project with the aim of delivering smaller,
incremental updates over time.

We encourage you to [read existing issues](https://github.com/hashicorp/terraform-ls/issues)
and/or [open new issue](https://github.com/hashicorp/terraform-ls/issues/new/choose)
if you experience a bug or have an idea for a feature.

## Disclaimer

This is not an officially supported HashiCorp product.

## Installation

```
go install .
```

This should produce a binary called `terraform-ls` in `$GOBIN/terraform-ls`.

Putting `$GOBIN` in your `$PATH` may save you from having to specify
absolute path to the binary.

## Usage

The most reasonable way you will interact with the language server
is through a client represented by an IDE, or a plugin of an IDE.

Please follow the [relevant guide for your IDE](./docs/USAGE.md).

## Credits

The implementation was inspired by:

 - [Julio Sueiras](https://github.com/juliosueiras) - particularly by his early version
    of the Language Server (https://github.com/juliosueiras)
 - [Martin Atkins](https://github.com/apparentlymart) (particularly the virtual filesystem)
