# Code Actions

The Terraform Language Server implements a set of Code Actions which perform different actions on the current document. These commands are typically code fixes to either refactor code, fix problems or to beautify/refactor code.

## Available Actions

### `source.formatAll.terraform`

The server will format a given document according to Terraform formatting conventions.

> *Important:* Disable `editor.formatOnSave` if using `source.formatAll.terraform`

The `source.formatAll.terraform` code action is is meant to be used instead of `editor.formatOnSave`, as it provides a [guarantee of order of execution](https://github.com/microsoft/vscode-docs/blob/71643d75d942e2c32cfd781c2b5322521775fb4a/release-notes/v1_44.md#explicit-ordering-for-editorcodeactionsonsave) based on the list provided. If you have both settings enabled, then your document will be formatted twice.

## Usage

### VS Code

To enable the format code action globally, set `source.formatAll.terraform` to *true* for the `editor.codeActionsOnSave` setting and set `editor.formatOnSave` to *false*.

```json
"editor.formatOnSave": false,
"editor.codeActionsOnSave": {
  "source.formatAll.terraform": true
},
"[terraform]": {
  "editor.defaultFormatter": "hashicorp.terraform",
}
```

If you would like `editor.formatOnSave` to be *true* for other extensions but *false* for the Terraform extension, you can configure your settings as follows:

```json
"editor.formatOnSave": true,
"editor.codeActionsOnSave": {
  "source.formatAll.terraform": true
},
"[terraform]": {
  "editor.defaultFormatter": "hashicorp.terraform",
  "editor.formatOnSave": false,
},
```

Alternatively, you can include all terraform related Code Actions inside the language specific setting if you prefer:

```json
"editor.formatOnSave": true,
"[terraform]": {
  "editor.defaultFormatter": "hashicorp.terraform",
  "editor.formatOnSave": false,
  "editor.codeActionsOnSave": {
    "source.formatAll.terraform": true
  },
},
```

## Developer Implementation Details

### Code Actions

A Code Action is an action that changes content in the active editor. Each Code Action is grouped into kinds that have a `command` and/or a series of `edits`. They are triggered either by the user or through events.

### Triggers

In VS Code, code action can be _invoked manually_ or _automatically_ based on the respective [CodeActionTriggerKind](https://code.visualstudio.com/api/references/vscode-api#CodeActionTriggerKind).

**Manually invoked** actions come from the contextual in-line💡 icon inside the editor, and are chosen by the user. The user can choose which action is invoked and *then* invoke it. However, in order for the client to display the contextual actions, the client requests LS to "pre-calculate" any actions relevant to the cursor position. Then, when the user selects the action in the UI, the client applies the `edits` or executes the `command` as provided by the server.

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
