
locals {
  number_local = 500
}

locals {
  include_resource_variable_2 = false
}

provider "aws" {
  alias = "this"
}

list "concept_pet" "name_1" {
  provider         = aws.this
  limit           = local.number_local
  include_resource = var.include_resource_variable
  count           = var.number_variable
  config {

  }
}
