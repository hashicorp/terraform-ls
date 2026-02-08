policy {
  consumer = terraform

  consumer_config {
    required_version = ">=1.12"
  }
}

locals {
  parent_value = core::getresources("test_resource", {
id = attrs.dependent
}).value
}

resource_policy "test_resource" "policy" {
filter = attrs.value == "child"

enforce {
condition     = local.parent_value == "parent"
error_message = "Child resource must link to a 'parent' resource."
}
}