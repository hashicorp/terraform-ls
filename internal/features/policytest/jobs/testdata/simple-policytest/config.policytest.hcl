policytest {
  targets = ["aws_instance.policy.hcl"]

  plugins {
    sample = {
      source = "./plugin/plugin_binary"
    }
  }
}


locals {
  instance_type = "t2.micro"
}

data "aws_ami" "ubuntu_ami" {
  attrs = {
    filter = [{
      name   = "image-id"
      values = ["ami-ubuntu-12345"]
    }]
    id           = "ami-ubuntu-12345"
    name         = "ubuntu-22.04-server-20250101"
    architecture = "x86_64"
    owner_id     = "099720109477"
  }
}


resource "aws_vpc" "main_vpc" {
  skip = true
  attrs = {
    id         = "vpc-12345678"
    cidr_block = "10.0.0.0/16"
    tags = {
      Name        = "main-vpc"
      Environment = "production"
    }
  }
}