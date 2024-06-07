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
  network_01_subnet_01 = "${var.network_01_name}-subnet-01"
  network_01_subnet_02 = "${var.network_01_name}-subnet-02"
  network_01_subnet_03 = "${var.network_01_name}-subnet-03"
  network_02_subnet_01 = "${var.network_02_name}-subnet-01"
  network_02_subnet_02 = "${var.network_02_name}-subnet-02"

  network_01_routes = [
    {
      name              = "${var.network_01_name}-egress-inet"
      description       = "route through IGW to access internet"
      destination_range = "0.0.0.0/0"
      tags              = "egress-inet"
      next_hop_internet = "true"
    },
  ]

  network_02_routes = [
    {
      name              = "${var.network_02_name}-egress-inet"
      description       = "route through IGW to access internet"
      destination_range = "0.0.0.0/0"
      tags              = "egress-inet"
      next_hop_internet = "true"
    },
    {
      name              = "${var.network_02_name}-testapp-proxy"
      description       = "route through proxy to reach app"
      destination_range = "10.50.10.0/24"
      tags              = "app-proxy"
      next_hop_ip       = "10.10.40.10"
    },
  ]
}

module "test-vpc-module-01" {
  source       = "../../"
  project_id   = var.project_id
  network_name = var.network_01_name

  subnets = [
    {
      subnet_name           = local.network_01_subnet_01
      subnet_ip             = "10.10.10.0/24"
      subnet_region         = "us-west1"
      subnet_private_access = "false"
      subnet_flow_logs      = "true"
    },
    {
      subnet_name           = local.network_01_subnet_02
      subnet_ip             = "10.10.20.0/24"
      subnet_region         = "us-west1"
      subnet_private_access = "false"
      subnet_flow_logs      = "true"
    },
    {
      subnet_name           = local.network_01_subnet_03
      subnet_ip             = "10.10.30.0/24"
      subnet_region         = "us-west1"
      subnet_private_access = "false"
      subnet_flow_logs      = "true"
    },
  ]

  secondary_ranges = {
    "${local.network_01_subnet_01}" = [
      {
        range_name    = "${local.network_01_subnet_01}-01"
        ip_cidr_range = "192.168.64.0/24"
      },
      {
        range_name    = "${local.network_01_subnet_01}-02"
        ip_cidr_range = "192.168.65.0/24"
      },
    ]

    "${local.network_01_subnet_02}" = [
      {
        range_name    = "${local.network_02_subnet_01}-01"
        ip_cidr_range = "192.168.74.0/24"
      },
    ]
  }

  routes = "${local.network_01_routes}"
}

module "test-vpc-module-02" {
  source       = "../../"
  project_id   = var.project_id
  network_name = var.network_02_name

  subnets = [
    {
      subnet_name           = "${local.network_02_subnet_01}"
      subnet_ip             = "10.10.40.0/24"
      subnet_region         = "us-west1"
      subnet_private_access = "false"
      subnet_flow_logs      = "true"
    },
    {
      subnet_name           = "${local.network_02_subnet_02}"
      subnet_ip             = "10.10.50.0/24"
      subnet_region         = "us-west1"
      subnet_private_access = "false"
      subnet_flow_logs      = "true"
    },
  ]

  secondary_ranges = {
    "${local.network_02_subnet_01}" = [
      {
        range_name    = "${local.network_02_subnet_02}-01"
        ip_cidr_range = "192.168.75.0/24"
      },
    ]
  }

  routes = local.network_02_routes
}
