# Installation

## Automatic Installation

Some editors have built-in logic to install and update the language server automatically, so you typically shouldn't need to worry about installation or updating of the server in these editors, as long as you use the linked extension.

 - Terraform VS Code extension [stable](https://marketplace.visualstudio.com/items?itemName=HashiCorp.terraform) / [preview](https://marketplace.visualstudio.com/items?itemName=HashiCorp.terraform-preview)
 - [Sublime Text LSP-terraform](https://packagecontrol.io/packages/LSP-terraform)

## Manual Installation

You can install the language server manually using one of the many package managers available or download an archive from the release page. After installation, follow the [install instructions for your IDE](./USAGE.md)

### Homebrew (macOS / Linux)

You can install via [Homebrew](https://brew.sh)

```
brew install terraform-ls
```

This tap only contains stable releases (i.e. no pre-releases).

### Linux

We support Debian & Ubuntu via apt and RHEL, CentOS, Fedora and Amazon Linux via RPM.

You can follow the instructions in the [Official Packaging Guide](https://www.hashicorp.com/official-packaging-guide) to install the server from the official HashiCorp-maintained repositories. The package name is `terraform-ls` in all repositories.

As documented in the Guide linked above, pre-releases are available through test repos.

### Other platforms

1. [Download for the latest version](https://releases.hashicorp.com/terraform-ls/)
  of the language server relevant for your operating system and architecture.
2. The language server is distributed as a single binary.
  Install it by unzipping it and moving it to a directory
  included in your system's `PATH`.
3. You can verify integrity by comparing the SHA256 checksums
  which are part of the release (called `terraform-ls_<VERSION>_SHA256SUMS`).
4. Check that you have installed the server correctly via `terraform-ls -v`.
  You should see the latest version printed to your terminal.
