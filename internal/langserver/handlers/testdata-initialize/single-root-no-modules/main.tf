resource "random_pet" "application" {
  count = var.count
  keepers = {
    unique = "unique"
  }
}

variable "count" {
  type = number
  default = 3
}

output "pet_count" {
  value = var.count
}
