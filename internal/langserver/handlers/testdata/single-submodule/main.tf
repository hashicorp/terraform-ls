module "gorilla-app" {
  source           = "./application"
  environment_name = "prod"
  app_prefix       = "protect-gorillas"
  instances        = 5
}
