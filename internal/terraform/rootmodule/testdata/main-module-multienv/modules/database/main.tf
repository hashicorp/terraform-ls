variable "environment_name" {
  type = string
}

variable "db_prefix" {
  type = string
}

variable "instances" {
  type = number
}

resource "random_pet" "database" {
  count = var.instances
  keepers = {
    unique = "${var.environment_name}-${var.db_prefix}"
  }
}
