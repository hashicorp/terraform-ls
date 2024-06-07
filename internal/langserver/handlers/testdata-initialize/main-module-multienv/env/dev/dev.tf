provider "random" {
  version = "~>2.0"
}

module "main" {
  source           = "../../main"
  environment_name = "dev"
  app_instances    = 1
  db_instances     = 1
}
