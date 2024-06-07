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
  net_data_users = compact(concat(
    var.service_project_owners,
    ["serviceAccount:${var.service_project_number}@cloudservices.gserviceaccount.com"]
  ))
}

module "net-vpc-shared" {
  source          = "../.."
  project_id      = var.host_project_id
  network_name    = var.network_name
  shared_vpc_host = true

  subnets = [
    {
      subnet_name   = "networking"
      subnet_ip     = "10.10.10.0/24"
      subnet_region = "europe-west1"
    },
    {
      subnet_name   = "data"
      subnet_ip     = "10.10.20.0/24"
      subnet_region = "europe-west1"
    },
  ]
}

module "net-svpc-access" {
  source              = "../../modules/fabric-net-svpc-access"
  host_project_id     = module.net-vpc-shared.project_id
  service_project_num = 1
  service_project_ids = [var.service_project_id]
  host_subnets        = ["data"]
  host_subnet_regions = ["europe-west1"]
  host_subnet_users = {
    data = join(",", local.net_data_users)
  }
}
