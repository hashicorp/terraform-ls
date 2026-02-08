# Main policy configuration block
policy {
  consumer = terraform

  consumer_config {
    # Require Terraform version >=1.12
    required_version = ">=1.12"
  }
}

locals {
  # Logic to fetch parent resources based on the 'dependent' attribute
  # parent_value will be used in the enforcement condition below
  parent_value = core::getresources("test_resource", {
id = attrs.dependent
}).value
}

# Enforcement policy for the 'test_resource' type
resource_policy "test_resource" "policy" {
# The 'attrs' object represents the resource being evaluated
filter = attrs.value == "child"

enforce {
condition     = local.parent_value == "parent"
error_message = "Child resource must link to a 'parent' resource."
}
}