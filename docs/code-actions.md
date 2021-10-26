# Code Actions

The Terraform Language Server implements a set of Code Actions which perform different actions on the current document. These commands are typically code fixes to either refactor code, fix problems or to beautify/refactor code.

## Available Actions

### User Code Actions

There are currently no Code Actions that can be invoked by the user yet.

### Automatic Code Actions

Automatic Code Actions happen when certain events are trigged, for example saving a document.

#### `source.formatAll.terraform`

The server will format a given document according to Terraform formatting conventions.

> *Important:* Disable `editor.formatOnSave` if using `source.formatAll.terraform`

The `source.formatAll.terraform` code action is is meant to be used instead of `editor.formatOnSave`, as it provides a guarantee of order of execution based on the list provided. If you have both settings enabled, then your document will be formatted twice.

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

### Code Action Events

A `Code Action` can have either an `invoke` trigger or an `automatic` [CodeActionTriggerKind](https://code.visualstudio.com/api/references/vscode-api#CodeActionTriggerKind).

`Invoked` actions come from the `lightbulb` UI inside the editor, and are chosen by the user. From the User POV, it appears that the user can choose which action is invoked from the UI and *then* it is invoked. This is not true. When the `lightbulb` UI is invoked, the LS receives a request for all supported code actions it can perform. The LS then performs all actions it can perform, then returns the `edits` or `commands` each action would perform. Then, when the user selects the action from the lightbulb UI, the client applies the `edits` or executes the `command` requested.

`Automatic` actions come from events like the `editor.codeActionsOnSave` setting. These usually do not give much choice to the user, they are either on or off, as they cannot accept user input. For example, formatting a document or removing simple style errors don't prompt for user action before or during execution.

### Code Action Types

Each `Code Action` has a [`CodeActionKind`](https://code.visualstudio.com/api/references/vscode-api#CodeActionKind). `Code Action Kinds` are a hierarchical list of identifiers separated by `.`. For example in `refactor.extract.function`: `refactor` is the trunk, `extract` is the branch, and `function` is the leaf. The branches and leaves are the point of intended customization, you add new branches and/or leaves for each type of function you perform. In this example, a new code action that operated on variables would be called `refactor.extract.variable`.

### Custom Code Action Types

Adding new roots or branches of the hierarchy is not emphasized or really encouraged. In most cases they are meant to be concepts that can be applied generically, not specifically to a certain language. For example, extracting a value to a variable is something common to most languages. You do not need a `refactor.extract.golang.variable` to extract a variable in Go, it still makes sense to use `refactor.extract.variable` whether in Go or in PowerShell.

Keeping to existing `kinds` also helps in registration of supported code actions. If you register support for `source.fixAll` instead of `source.fixAll.languageId`, then a user that has `source.fixAll` in their `codeActionsOnSave` does not need to add specific lines for your extension, all supported extensions are called on save. Another reason is VS Code won't list your custom supported code actions inside `editor.codeActionsOnSave`, and there isn't a way for the extension to get them there. The user will have to consult your documentation to find out what actions are supported and add them without intellisense.

A reason to add custom `code action kinds` is if your action is sufficiently different from an existing base action. For example, your extension wants to support formatting the current file on save through the `editor.codeActionsOnSave`. The definition of `source.fixAll` is `Fix all actions automatically fix errors that have a clear fix that do not require user input`. Your extension only supports formatting syntax errors, not fixing simple errors, so your action does not quite fit the definition for `source.fixAll`. You could introduce a new code action kind called `source.formallAll`, which is meant to only format syntax errors. A user then can add both `source.fixAll` and `source.formatAll` to their settings and your extension would run `source.formatAll`. Other extensions would run `source.fixAll` but not `source.formatAll` as they do not use your custom code action kind. 

So while you were able to add your custom action kind, it's only discoverable by your documentation and only usable if the user adds your specific kind to their settings, not a general kind. This may be an important distinction to your extension, it may be not, it’s up to you to decide.

### Execution of Code Actions

A request can have zero or more code actions to perform and the[ LS is responsible for processing](https://github.com/microsoft/language-server-protocol/issues/970) all requested actions. This means several complicated operations need to happen on the LS. The client will send either a list of code actions to execute or an empty list.

An empty list means execute anything supported and return the list of edits to the client. This often comes from the `CodeActionTriggerKind.Invoke` used by the lightbulb provider. This is the easiest situation for the LS to choose what to do. The LS has a list of supported actions it can execute, so it executes and returns the edits. However, the list of supported actions should not include anything that either requires user input or makes a change that could introduce an error (creating files, changing code structure, etc). 

A list of actions means to execute anything in that list. Which sounds simple, but is actually ‘execute the hierarchy of a code action from these actions in this list. For example, if a client sends a `refactor` request the LS should return all of the possible refactor actions it can execute (refactor.extract, refactor.inline, refactor.rewrite, etc), and the user can choose which to execute. A client that sends `refactor.extract.method` would receive just that code action from the LS. So a request with one action could return ten results, or just one.

Additionally a client could send a list of actions that mean the same execution on the LS. Clients are supposed to self filter, but implementations vary and there may be reasons they can’t filter client side. For (a contrived) example, a client sends `source.fixAll` and `source.fixAll.eslint`. The LS would have to inspect and know that those two actions mean the same thing, and only run one action. It should favor the most specific (in this case source.fixAll.eslint) but this is not well defined so implementations may vary.

There is also a semi-documented time limit for returning a response for any of these requests. If the LS takes too long to process and return, the client will either (depending on how recent a VS Code version, or what client implementation) give up and not do anything leaving the user with no visual information on what is going on, or hang with a progress indicator. The user can increase the timeout, but that is considered an anti-pattern.
