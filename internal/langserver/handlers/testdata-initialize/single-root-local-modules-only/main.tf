module "first" {
    source = "./alpha"
}

module "second" {
    source = "./alpha"
}

module "three" {
    source = "./beta"
}
