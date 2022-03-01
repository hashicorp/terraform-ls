# Terraform Language Server

The official [Terraform](https://www.terraform.io) language server (`terraform-ls`) maintained by [HashiCorp](https://www.hashicorp.com) provides IDE features to any [LSP](https://microsoft.github.io/language-server-protocol/)-compatible editor.

## Current Status

Not all language features (from LSP's or any other perspective) are available
at the time of writing, but this is an active project with the aim of delivering
smaller, incremental updates over time.

We encourage you to [browse existing issues](https://github.com/hashicorp/terraform-ls/issues)
and/or [open new issue](https://github.com/hashicorp/terraform-ls/issues/new/choose)
if you experience a bug or have an idea for a feature.

## Stability

We aim to communicate our intentions regarding breaking changes via [semver](https://semver.org). Relatedly we may use pre-releases, such as `MAJOR.MINOR.PATCH-beta1` to gather early feedback on certain features and changes.

We ask that you [report any bugs](https://github.com/hashicorp/terraform-ls/issues/new/choose) in any versions but especially in pre-releases, if you decide to use them.

## Installation

Some editors have built-in logic to install and update the language server automatically, so you may not need to worry about installation or updating of the server.

Read the [installation page](./docs/installation.md) for installation instructions.

## Usage

The most reasonable way you will interact with the language server
is through a client represented by an IDE, or a plugin of an IDE.

Please follow the [relevant guide for your IDE](./docs/USAGE.md).

## Credits

- [Martin Atkins](https://github.com/apparentlymart) - particularly the virtual filesystem
- [Zhe Cheng](https://github.com/njuCZ) - research, design, prototyping assistance
- [Julio Sueiras](https://github.com/juliosueiras) - particularly his [language server implementation](https://github.com/juliosueiras/terraform-lsp)

## Telemetry

The server will collect data only if the _client_ requests it during initialization. Telemetry is opt-in by default.

[Read more about telemetry](./docs/telemetry.md).

## `terraform-ls` VS `terraform-lsp`

Both HashiCorp and [the maintainer](https://github.com/juliosueiras) of [`terraform-lsp`](https://github.com/juliosueiras/terraform-lsp)
expressed interest in collaborating on a language server and are working
towards a _long-term_ goal of a single stable and feature-complete implementation.

For the time being both projects continue to exist, giving users the choice:

- `terraform-ls` providing
  - overall stability (by relying only on public APIs)
  - compatibility with any provider and any Terraform `>=0.12.0`
  - currently less features
    - due to project being younger and relying on public APIs which may not
      offer the same functionality yet

- `terraform-lsp` providing
  - currently more features
  - compatibility with a single particular Terraform (`0.12.20` at time of writing)
    - configs designed for other `0.12` versions may work, but interpretation may be inaccurate
  - less stability (due to reliance on Terraform's own internal packages)
