schema = "1"

project "terraform-ls" {
  // team is currently unused and has no meaning
  // but is required to be non-empty by CRT orchestator
  team = "_UNUSED_"
  slack {
    notification_channel = "C02AGQXCAF5" // # feed-tf-editor-experience
  }
  github {
    organization = "hashicorp"
    repository = "terraform-ls"
    release_branches = ["main"]
  }
}

event "build" {
  action "build" {
    organization = "hashicorp"
    repository = "terraform-ls"
    workflow = "build"
  }
}

event "prepare" {
  depends = ["build"]

  action "prepare" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "prepare"
    depends      = ["build"]
  }

  notification {
    on = "fail"
  }
}

## These are promotion and post-publish events
## they should be added to the end of the file after the verify event stanza.

event "trigger-staging" {
// This event is dispatched by the bob trigger-promotion command
// and is required - do not delete.
}

event "promote-staging" {
  depends = ["trigger-staging"]
  action "promote-staging" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "promote-staging"
    config = "release-metadata.hcl"
  }

  notification {
    on = "always"
  }
}

event "trigger-production" {
// This event is dispatched by the bob trigger-promotion command
// and is required - do not delete.
}

event "promote-production" {
  depends = ["trigger-production"]
  action "promote-production" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "promote-production"
  }

  notification {
    on = "always"
  }
}

event "promote-production-packaging" {
  depends = ["promote-production"]
  action "promote-production-packaging" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "promote-production-packaging"
  }

  notification {
    on = "always"
  }
}