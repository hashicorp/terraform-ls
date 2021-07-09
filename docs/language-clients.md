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

### Multiple Folders

Language server supports multiple folders natively from version `0.19`.

Client is expected to always launch a single instance of the server and check for
[`workspace.workspaceFolders.supported`](https://microsoft.github.io/language-server-protocol/specifications/specification-3-17/#workspaceFoldersServerCapabilities) server capability, and then:

 - launch any more instances (_one instance per folder_) if multiple folders are _not supported_
 - avoid launching any more instances if multiple folders _are supported_

It is assumed that paths to these folders will be provided as part of `workspaceFolders`
in the `initialize` request per LSP.

## Code Lens

### Reference Counts (opt-in)

The server implements an opt-in code lens which displays number of references
to any "root level" targettable block or attribute, such as local value,
variable, resource etc.

LSP has not standardized client-side command IDs nor does it provide mechanism
for negotiating what the right command ID is and whether it's available.
This is why **client has to opt-in by providing a command ID** in experimental
client capabilities.

For example:

```json
{
    "capabilities": {
        "experimental": {
            "showReferencesCommandId": "client.showReferences"
        }
    }
}
```

This enables the code lens.

The client-side command is executed with 2 arguments (position, reference context):

```json
[
    {
        "line": 0,
        "character": 8
    },
    {
        "includeDeclaration": false
    }
]
```

These arguments are to be passed by the client to a subsequent [`textDocument/references`](https://microsoft.github.io/language-server-protocol/specifications/specification-current/#textDocument_references)
request back to the server to obtain the list of references relevant to
that position and finally display received references in the editor.

See [example implementation in the Terraform VS Code extension](https://github.com/hashicorp/vscode-terraform/pull/686).


## Custom Commands

Clients are encouraged to implement custom commands
in a command palette or similar functionality.

See [./commands.md](./commands.md) for more.
