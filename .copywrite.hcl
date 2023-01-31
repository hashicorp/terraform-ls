schema_version = 1

project {
  license        = "MPL-2.0"
  copyright_year = 2020

  header_ignore = [
    "**/testdata/**",
    "internal/schemas/providers.tf",
    "internal/schemas/data/**",
    "internal/schemas/gen-workspace/**",
    "internal/schemas/tf-plugin-cache/**",
  ]
}
