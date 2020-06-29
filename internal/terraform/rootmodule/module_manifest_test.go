package rootmodule

const moduleManifestRecord_external = `{
    "Key": "web_server_sg",
    "Source": "terraform-aws-modules/security-group/aws//modules/http-80",
    "Version": "3.10.0",
    "Dir": ".terraform/modules/web_server_sg/terraform-aws-security-group-3.10.0/modules/http-80"
}`

const moduleManifestRecord_externalDirtyPath = `{
    "Key": "web_server_sg",
    "Source": "terraform-aws-modules/security-group/aws//modules/http-80",
    "Version": "3.10.0",
    "Dir": ".terraform/modules/web_server_sg/terraform-aws-security-group-3.10.0/modules/something/../http-80"
}`

const moduleManifestRecord_local = `{
    "Key": "local",
    "Source": "./nested/path",
    "Dir": "nested/path"
}`

const moduleManifestRecord_root = `{
    "Key": "",
    "Source": "",
    "Dir": "."
}`
