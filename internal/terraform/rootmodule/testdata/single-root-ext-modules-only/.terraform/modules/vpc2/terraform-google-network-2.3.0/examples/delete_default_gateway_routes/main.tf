/**
 * Copyright 2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

provider "google" {
  version = "~> 3.3.0"
}

provider "null" {
  version = "~> 2.1"
}

locals {
  subnet_01 = "${var.network_name}-subnet-01"
}

module "test-vpc-module" {
  source                                 = "../../"
  project_id                             = var.project_id
  network_name                           = var.network_name
  delete_default_internet_gateway_routes = "true"

  subnets = [
    {
      subnet_name   = local.subnet_01
      subnet_ip     = "10.20.30.0/24"
      subnet_region = "us-west1"
    },
  ]
}
