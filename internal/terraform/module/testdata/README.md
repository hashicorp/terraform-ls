# Tested Terraform Hierarchies

This directory contains different hierarchies of root modules
which the language server supports and is tested against.

## Single Root

 - `single-root-ext-modules-only`
 - `single-root-local-and-ext-modules`
 - `single-root-local-modules-only`
 - `single-root-no-modules`

## Nested Single Root

 - `nested-single-root-no-modules`
 - `nested-single-root-ext-modules-only`
 - `nested-single-root-local-modules-down`
 - `nested-single-root-local-modules-up`

## Multiple Roots

 - `main-module-multienv` - https://dev.to/piotrgwiazda/main-module-approach-for-handling-multiple-environments-in-terraform-1oln
 - `multi-root-no-modules`
 - `multi-root-local-modules-down`
 - `multi-root-local-modules-up` - e.g. https://github.com/terraform-aws-modules/terraform-aws-security-group

## Uninitialized Root

 - `uninitialized-root`
