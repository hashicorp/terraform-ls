provider "random" {
  version = "~>2.0"
}

module "main" {
  source           = "../../main"
  environment_name = "staging"
  app_instances    = 2
  db_instances     = 1
}
