# Usage of Terraform Language Server

This guide assumes you have installed the server by following instructions
in the [README.md](../README.md) if that is applicable to your client
(i.e. if the client doesn't download the server itself).

Instructions for popular IDEs are below and pull requests
for updates or addition of more IDEs are welcomed.

## Sublime Text 2

 - Install the [LSP package](https://github.com/sublimelsp/LSP#installation)
 - Add the following snippet to the LSP settings' `clients`:

```json
"terraform": {
  "command": ["terraform-ls", "serve"],
  "enabled": true,
  "languageId": "terraform",
  "scopes": ["source.terraform"],
  "syntaxes": ["Packages/Terraform/Terraform.sublime-syntax"]
}
```
