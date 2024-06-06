module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "18.23.0"

}

module "ec" {
  source  = "puppetlabs/deployment/ec"
  version = "0.0.8"

}
