---
name: Bug report
about: Let us know about an unexpected error, a crash, or an incorrect behavior.

---

### Server Version
<!--
Run `terraform-ls --version` to show the version, and paste the result between the ``` marks below.
If you are not running the latest version, please try upgrading because your issue may have already been fixed.
-->
```

```

### Terraform Version
<!--
Run `terraform -v to show the version, and paste the result between the ``` marks below.
Make sure you are running the same binary that the language server would normally pick up from $PATH
if you have more than one version installed on your system
-->
```

```

### Client Version
<!--
Please share what IDE and/or plugin which interacts with the server
e.g. Sublime Text (LSP plugin) v0.9.7
-->
```

```

### Terraform Configuration Files
<!--
Paste the relevant parts of your Terraform configuration between the ``` marks below.

For large Terraform configs, please use a service like Dropbox and share a link to the ZIP file.
For security, you can also encrypt the files using HashiCorp's GPG public key published at
https://www.hashicorp.com/security#secure-communications
-->

```hcl

```

### Log Output
<!--
Full debug output can be obtained by launching server with particular flag (e.g. -log-file)
Please follow instructions in docs/TROUBLESHOOTING.md

Please create a GitHub Gist containing the debug output. Please do *NOT* paste
the debug output in the issue, since it may be long.

Debug output may contain sensitive information. Please review it before posting publicly, and if you are concerned feel free to encrypt the files using HashiCorp's GPG public key published at
https://www.hashicorp.com/security#secure-communications
-->

### Expected Behavior
<!-- What should have happened? -->

### Actual Behavior
<!-- What actually happened? -->

### Steps to Reproduce
<!--
Please list the full steps required to reproduce the issue, for example:
1. Open a folder in IDE XYZ
2. Open file example.tf from that folder
3. Trigger autocompletion on line 5, column 1 (1-indexed)
-->
