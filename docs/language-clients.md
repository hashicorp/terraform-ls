# Language Client Implementation Notes

This document contains notes for language client developers.

## Language IDs

The following file types are currently supported and language IDs expected:

- `terraform` - standard `*.tf` config files
- `terraform-vars` - variable files (`*.tfvars`)
- `terraform-stack` - standard `*.tfstack.hcl` files
- `terraform-deploy` - standard `*.tfstack.hcl` files

Client can choose to highlight other files locally, but such other files
must **not** be send to the server as the server isn't equipped to handle those.

Clients specifically should **not** send `*.tf.json`, `*.tfvars.json` nor
Packer HCL config nor any other HCL config files as the server is not
equipped to handle these file types.

## Configuration

Unless the client allows the end-user to pass arbitrary config options (e.g.
generic Sublime Text LSP package without Terraform LSP package), the client
should expose configuration as per [SETTINGS.md](./SETTINGS.md).

Client should match the option names exactly, and if possible match the
underlying data structures too. i.e. if a field is documented as `ignoreDirectoryNames`,
it should be exposed as `ignoreDirectoryNames`, **not** ~`IgnoreDirectoryNames`~,
or ~`ignore_directory_names`~. This is to avoid user confusion when the server
refers to any config option in informative, warning, or error messages.

Client may use a flat structure using the `.` (single dot) as a separator between
the object name and option nested under it, such as `{ "foo.bar": "..." }` instead
of `{ "foo": { "bar": "..." } }`. This is acceptable in situations when using
objects is not possible or practical (e.g. VS Code wouldn't display objects
in the Settings UI).

The server will generally refer to options using the `.` address, for simplicity
and avoidance of doubts.

## Watched Files

The server (`>= 0.28.0`) supports `workspace/didChangeWatchedFiles` notifications.
This allows IntelliSense to remain accurate e.g. when switching branches in VCS
or when there are any other changes made to these files outside the editor.

If the client implements file watcher, it should watch for any changes
in `**/*.tf`, `**/*.tfvars`, `**/*.tfstack.hcl` and `**/*.tfstack.hcl` files in the workspace.

Client should **not** send changes for any other files.

## Syntax Highlighting

Read more about how we recommend Terraform files to be highlighted in [syntax-highlighting.md](./syntax-highlighting.md).

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

## Code Actions

The server implements a set of opt-in code actions which perform different actions for the user. The code action request is sent from the client to the server to compute commands for a given text document and range. These commands are typically code fixes to either fix problems or to beautify/refactor code.

See [code-actions](./code-actions.md) for a list of supported code actions.

A Code Action is an action that changes content in the active editor. Each Code Action is grouped into kinds that have a `command` and/or a series of `edits`. They are triggered either by the user or through events.

> Documentation for code actions outside of VS Code is unfortunately very limited beyond description of the LSP methods. VS Code internally makes certain assumptions. We follow these assumptions (as documented below) and we recommend other clients to follow these assumptions for best experience, unless/until LSP documentation recommends otherwise.

### Triggers

In VS Code, code action can be _invoked manually_ or _automatically_ based on the respective [CodeActionTriggerKind](https://code.visualstudio.com/api/references/vscode-api#CodeActionTriggerKind).

**Manually invoked** actions come from the contextual in-line💡 icon inside the editor, and are chosen by the user. The user can choose which action is invoked and _then_ invoke it. However, in order for the client to display the contextual actions, the client requests LS to "pre-calculate" any actions relevant to the cursor position. Then, when the user selects the action in the UI, the client applies the `edits` or executes the `command` as provided by the server.

**Automatically triggered** actions come from events such as "on save", as configured via the `editor.codeActionsOnSave` setting. These usually do not give much choice to the user, they are either on or off, as they cannot accept user input. For example, formatting a document or removing simple style errors don't prompt for user action before or during execution.

### Kinds

Each `Code Action` has a [`CodeActionKind`](https://code.visualstudio.com/api/references/vscode-api#CodeActionKind). `Code Action Kinds` are a hierarchical list of identifiers separated by `.`. For example in `refactor.extract.function`: `refactor` is the trunk, `extract` is the branch, and `function` is the leaf. The branches and leaves are the point of intended customization, you add new branches and/or leaves for each type of function you perform. In this example, a new code action that operated on variables would be called `refactor.extract.variable`.

#### Custom Kinds

Adding new roots or branches of the hierarchy is not emphasized or really encouraged. In most cases they are meant to be concepts that can be applied generically, not specifically to a certain language. For example, extracting a value to a variable is something common to most languages. You do not need a `refactor.extract.golang.variable` to extract a variable in Go, it still makes sense to use `refactor.extract.variable` whether in Go or in PowerShell.

Keeping to existing `kinds` also helps in registration of supported code actions. If you register support for `source.fixAll` instead of `source.fixAll.languageId`, then a user that has `source.fixAll` in their `codeActionsOnSave` does not need to add specific lines for your extension, all supported extensions are called on save. Another reason is VS Code won't list your custom supported code actions inside `editor.codeActionsOnSave`, and there isn't a way for the extension to get them there. The user will have to consult your documentation to find out what actions are supported and add them without intellisense.

A reason to add custom _kinds_ is if the action is sufficiently different from an existing base action. For example, formatting of the current file on save. The interpretation of `source.fixAll` is to _apply any/all actions that address an existing diagnostic and have a clear fix that does not require user input_. Formatting therefore doesn't fit the interpretation of `source.fixAll`.

A custom kind `source.formatAll.terraform` may format code. A user can request both `source.fixAll` and `source.formatAll.terraform` via their editor/client settings and the server would run `source.formatAll.terraform` only. Other servers may run `source.fixAll` but not `source.formatAll.terraform`, assuming they do not support that custom code action kind.

Unlike generic kinds, custom ones are only discoverable in server-specific documentation and only relevant to the server.

### Execution of Code Actions

A request can have zero or more code actions to perform and the [LS is responsible for processing](https://github.com/microsoft/language-server-protocol/issues/970) all requested actions. The client will send a list of any code actions to execute (which may also be empty).

An empty list means execute anything supported and return the list of edits to the client. This often comes from _manually invoked_ actions by the user. This is the easiest situation for the LS to choose what to do. The LS has a list of supported actions it can execute, so it executes and returns the edits. However, such actions will not include any that either require user input or make a change that could introduce an error (creating files, changing code structure, etc).

A list of actions means to execute anything in that list, which is actually interpreted as _execute the hierarchy of a code action from these actions in the list_. For example, if a client requests a `refactor` action the LS should return all of the possible `refactor` actions it can execute (`refactor.extract`, `refactor.inline`, `refactor.rewrite`, etc.), and the user can choose which to execute. A client that sends `refactor.extract.method` would receive just that single code action from the LS. So a request with one action could return ten results, or just one.

Clients are expected to filter actions to only send unique ones. For example, the user may have configured both `source.fixAll` and `source.fixAll.eslint` which are equivalent from the perspective of a single server (eslint). The client should favour the most specific (in this case `source.fixAll.eslint`) if possible, such that the server doesn't have to perform any de-duplication and the same action doesn't run multiple times.

Clients may also impose a timeout for returning response for any of these requests. If the LS takes too long to process and return an action, the client may either give up and not do anything, or (preferably) display a progress indicator. This timeout may be configurable by the user, but ideally the default one is sufficient.

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

## Telemetry

See [./telemetry.md](./telemetry.md).
