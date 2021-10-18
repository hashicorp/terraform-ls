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
