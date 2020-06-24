module "first" {
    source = "./alpha"
}

module "second" {
    source = "./beta"
}

module "three" {
    source = "./alpha"
}

module "four" {
    source  = "terraform-google-modules/network/google"
    version = "~> 2.3"

    project_id   = "1234567891234567"
    network_name = "example-first"
    routing_mode = "GLOBAL"

    subnets = [
        {
            subnet_name           = "subnet-a-01"
            subnet_ip             = "10.10.10.0/24"
            subnet_region         = "us-west1"
        }
    ]
}

module "five" {
    source  = "terraform-google-modules/network/google"
    version = "~> 2.3"

    project_id   = "1234567891234567"
    network_name = "example-second"
    routing_mode = "GLOBAL"

    subnets = [
        {
            subnet_name           = "subnet-b-01"
            subnet_ip             = "10.20.10.0/24"
            subnet_region         = "us-west1"
        }
    ]
}
