# Language Client Implementation Notes

This document contains notes for language client developers.

## Language IDs

The following file types are currently supported and language IDs expected:

 - `terraform` - standard `*.tf` config files
 - `terraform-vars` - variable files (`*.tfvars`)

Client can choose to highlight other files locally, but such other files
must **not** be send to the server as the server isn't equipped to handle those.

Clients specifically should **not** send `*.tf.json`, `*.tfvars.json` nor
Packer HCL config nor any other HCL config files as the server is not
equipped to handle these file types.

### Internal parser

The server expects clients to use standard text synchronization LSP methods
for synchronizing the above supported files.

Server will itself parse the whole module in order to provide completion/hover
and referencing throughout the module, not just within opened files.

As a result the server maintains an "overlay virtual filesystem" for any files
that client sends via LSP and where appropriate such files are treated as
the main source of truth, so that functionality can be provided even before
files are saved to disk.

## Use Client/Server Capabilities

Please always make sure that your client reads and reflects
[`ServerCapabilities`](https://microsoft.github.io/language-server-protocol/specifications/specification-3-17/#serverCapabilities)
and never makes blind assumptions about what is or is not supported.

The server will always read [`ClientCapabilities`](https://microsoft.github.io/language-server-protocol/specifications/specification-3-17/#clientCapabilities)
and make decisions about whether to provide any LSP features
based on those capabilities, so make sure these are accurate.

For example the server will not provide completion snippets unless the client
explicitly communicates it supports them via [`CompletionClientCapabilities`](https://microsoft.github.io/language-server-protocol/specifications/specification-3-17/#completionClientCapabilities).

## Custom Commands

Clients are encouraged to implement custom commands
in a command palette or similar functionality.

See [./commands.md](./commands.md) for more.
