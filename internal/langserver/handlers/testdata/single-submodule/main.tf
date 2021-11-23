module "gorilla-app" {
  source           = "./application"
  environment_name = "prod"
  app_prefix       = "protect-gorillas"
  instances        = var.instance_count
}

variable "instance_count" {
  default = 5
}
