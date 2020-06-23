resource "random_pet" "application" {
  count = 3
  keepers = {
    unique = "unique"
  }
}
