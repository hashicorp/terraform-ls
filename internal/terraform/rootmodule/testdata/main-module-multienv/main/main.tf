variable "environment_name" {
  type = string
}

variable "app_instances" {
  type = number
}

variable "db_instances" {
  type = number
}

module "db" {
  source           = "../modules/database"
  environment_name = var.environment_name
  app_prefix       = "foxtrot"
  instances        = var.db_instances
}

module "gorilla-app" {
  source           = "../modules/application"
  environment_name = var.environment_name
  app_prefix       = "protect-gorillas"
  instances        = var.app_instances
}
