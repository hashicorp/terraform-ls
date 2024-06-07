provider "aws" {
  version = "~> 2.0"
  region  = "us-east-1"
}

resource "aws_vpc" "example" {
  cidr_block = "10.0.0.0/16"
}
