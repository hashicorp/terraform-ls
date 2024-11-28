variable {
  type = string
}

locals {
  test = 1
}

provider "aws" {
  region = var.region
}
