provider "random" {
  version = "~>2.0"
}

module "main" {
  source           = "../../main"
  environment_name = "prod"
  app_instances    = 5
  db_instances     = 3
}
