# Features of the server

## LSP methods

The LSP is a relatively extensive protocol with many features/methods, not all of which are implemented and not all of which are relevant for Terraform. The following matrix should provide some visibility into the current and future state.

It's important to note that ✅ does **not** imply the functionality is _fully_ implemented (some features have various opt-in capabilities), just that the method is in use by the server. You can filter any known open issues by the relevant label, e.g. [`textDocument/completion` issues](https://github.com/hashicorp/terraform-ls/issues?q=is%3Aopen+is%3Aissue+label%3AtextDocument%2Fcompletion) and open new issues for any method which you would like to be implemented.

### Requests

| LSP method | Implemented | Note |
| :---       |    :----:   | :--- |
| callHierarchy/incomingCalls | ❌ | |
| callHierarchy/outgoingCalls | ❌ | |
| client/registerCapability | ❌ | |
| client/unregisterCapability | ❌ | |
| codeAction/resolve | ❌ | |
| codeLens/resolve | ❌ | |
| completionItem/resolve | ✅ | |
| documentLink/resolve | ❌ | |
| initialize | ✅ | |
| inlayHint/resolve | ❌ | |
| shutdown | ✅ | |
| textDocument/codeAction | ✅ | See [code-actions.md](https://github.com/hashicorp/terraform-ls/blob/main/docs/code-actions.md) |
| textDocument/codeLens | ✅ | See [Code Lens section](https://github.com/hashicorp/terraform-ls/blob/main/docs/language-clients.md#code-lens) |
| textDocument/colorPresentation | ❌ | Not relevant |
| textDocument/completion | ✅ | |
| textDocument/declaration | ✅ | |
| textDocument/definition | ✅ | |
| textDocument/diagnostic | ❌ | |
| textDocument/documentColor | ❌ | Not relevant |
| textDocument/documentHighlight | ❌ | |
| textDocument/documentLink | ✅ | |
| textDocument/documentSymbol | ✅ | |
| textDocument/foldingRange | ❌ | |
| textDocument/formatting | ✅ | |
| textDocument/hover | ✅ | |
| textDocument/implementation | ❌ | |
| textDocument/inlayHint | ❌ | |
| textDocument/inlineValue | ❌ | |
| textDocument/linkedEditingRange | ❌ | |
| textDocument/moniker | ❌ | |
| textDocument/onTypeFormatting | ❌ | |
| textDocument/prepareCallHierarchy | ❌ | |
| textDocument/prepareRename | ❌ | |
| textDocument/prepareTypeHierarchy | ❌ | |
| textDocument/rangeFormatting | ❌ | |
| textDocument/references | ✅ | |
| textDocument/rename | ❌ | |
| textDocument/selectionRange | ❌ | |
| textDocument/semanticTokens/full | ✅ | See [syntax-highlighting.md](https://github.com/hashicorp/terraform-ls/blob/main/docs/syntax-highlighting.md#semantic-tokens) |
| textDocument/semanticTokens/full/delta | ❌ | |
| textDocument/semanticTokens/range | ❌ | |
| textDocument/signatureHelp | ✅ | |
| textDocument/typeDefinition | ❌ | |
| textDocument/willSaveWaitUntil | ❌ | |
| typeHierarchy/subtypes | ❌ | |
| typeHierarchy/supertypes | ❌ | |
| window/showDocument | ❌ | |
| window/showMessageRequest | ✅ | |
| window/workDoneProgress/create | ❌ | |
| workspace/applyEdit | ❌ | |
| workspace/codeLens/refresh | ✅ | |
| workspace/configuration | ❌ | |
| workspace/diagnostic | ❌ | |
| workspace/diagnostic/refresh | ❌ | |
| workspace/executeCommand | ✅ | See [commands.md](https://github.com/hashicorp/terraform-ls/blob/main/docs/commands.md) |
| workspace/inlayHint/refresh | ❌ | |
| workspace/inlineValue/refresh | ❌ | |
| workspace/semanticTokens/refresh | ✅ | See [syntax-highlighting.md](https://github.com/hashicorp/terraform-ls/blob/main/docs/syntax-highlighting.md#semantic-tokens) |
| workspace/symbol | ✅ | |
| workspace/willCreateFiles | ❌ | |
| workspace/willDeleteFiles | ❌ | |
| workspace/willRenameFiles | ❌ | |
| workspace/workspaceFolders | ✅ | |
| workspaceSymbol/resolve | ❌ | |

List of methods sourced via
```sh
curl -s https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/metaModel/metaModel.json | jq -r '.requests[].method' | sort
```

### Notifications

| LSP method | Implemented | Note |
| :---       |    :----:   | :--- |
| $/cancelRequest | ✅ | |
| $/logTrace | ❌ | |
| $/progress | ✅ | |
| $/setTrace | ❌ | |
| exit | ✅ | |
| initialized | ✅ | |
| notebookDocument/didChange | ❌ | |
| notebookDocument/didClose | ❌ | |
| notebookDocument/didOpen | ❌ | |
| notebookDocument/didSave | ❌ | |
| telemetry/event | ✅ | See [telemetry.md](https://github.com/hashicorp/terraform-ls/blob/main/docs/telemetry.md) |
| textDocument/didChange | ✅ | |
| textDocument/didClose | ✅ | |
| textDocument/didOpen | ✅ | |
| textDocument/didSave | ✅ | |
| textDocument/publishDiagnostics | ✅ | |
| textDocument/willSave | ❌ | |
| window/logMessage | ❌ | |
| window/showMessage | ✅ | |
| window/workDoneProgress/cancel | ❌ | |
| workspace/didChangeConfiguration | ❌ | |
| workspace/didChangeWatchedFiles | ✅ | See [Watched Files section](https://github.com/hashicorp/terraform-ls/blob/main/docs/language-clients.md#watched-files) |
| workspace/didChangeWorkspaceFolders | ✅ | |
| workspace/didCreateFiles | ❌ | |
| workspace/didDeleteFiles | ❌ | |
| workspace/didRenameFiles | ❌ | |

List of methods sourced via
```sh
curl -s https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/metaModel/metaModel.json | jq -r '.requests[].method' | sort
```
