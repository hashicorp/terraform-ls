# Terraform Language Server

The official [Terraform](https://www.terraform.io) language server (`terraform-ls`) maintained by [HashiCorp](https://www.hashicorp.com) provides IDE features to any [LSP](https://microsoft.github.io/language-server-protocol/)-compatible editor.

## Current Status

Not all language features (from LSP's or any other perspective) are available
at the time of writing, but this is an active project with the aim of delivering
smaller, incremental updates over time. You can review [the LSP feature matrix](./docs/features.md).

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

## Contributing

Please refer to [.github/CONTRIBUTING.md](.github/CONTRIBUTING.md) for more information on how to contribute to this project.

## Credits

- [Martin Atkins](https://github.com/apparentlymart) - particularly the virtual filesystem
- [Zhe Cheng](https://github.com/njuCZ) - research, design, prototyping assistance
- [Julio Sueiras](https://github.com/juliosueiras) - particularly his [language server implementation](https://github.com/juliosueiras/terraform-lsp)

## Telemetry

The server will collect data only if the _client_ requests it during initialization. Telemetry is opt-in by default.

[Read more about telemetry](./docs/telemetry.md).