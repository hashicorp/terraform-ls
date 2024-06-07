variable "environment_name" {
  type = string
}

variable "app_prefix" {
  type = string
}

variable "instances" {
  type = number
}

resource "random_pet" "application" {
  count = var.instances
  keepers = {
    unique = "${var.environment_name}-${var.app_prefix}"
  }
}
