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
  version = "~> 2.19.0"
}

provider "google-beta" {
  version = "~> 2.19.0"
}

provider "null" {
  version = "~> 2.1"
}

module "vpc" {
  source       = "../../modules/vpc"
  network_name = var.network_name
  project_id   = var.project_id
}

module "subnets" {
  source       = "../../modules/subnets-beta"
  project_id   = var.project_id
  network_name = module.vpc.network_name

  subnets = [
    {
      subnet_name   = "${var.network_name}-subnet"
      subnet_ip     = "10.10.10.0/24"
      subnet_region = "us-west1"
    },
    {
      subnet_name   = "${var.network_name}-subnet-01"
      subnet_ip     = "10.20.10.0/24"
      subnet_region = "us-west1"
      purpose       = "INTERNAL_HTTPS_LOAD_BALANCER"
      role          = "ACTIVE"
    }
  ]
}

module "subnets-backup" {
  source       = "../../modules/subnets-beta"
  project_id   = var.project_id
  network_name = module.vpc.network_name

  subnets = [
    {
      subnet_name   = "${var.network_name}-subnet-02"
      subnet_ip     = "10.20.20.0/24"
      subnet_region = "us-west1"
      purpose       = "INTERNAL_HTTPS_LOAD_BALANCER"
      role          = "BACKUP"
    }
  ]

  module_depends_on = [module.subnets.subnets]
}

resource "google_compute_health_check" "this" {
  project            = var.project_id
  name               = "${var.network_name}-test"
  check_interval_sec = 1
  timeout_sec        = 1

  tcp_health_check {
    port = "80"
  }
}

resource "google_compute_region_backend_service" "this" {
  project       = var.project_id
  name          = "${var.network_name}-test"
  region        = "us-west1"
  health_checks = [google_compute_health_check.this.self_link]
}

resource "google_compute_forwarding_rule" "this" {
  project = var.project_id
  name    = "${var.network_name}-fw-role"

  network               = module.vpc.network_name
  subnetwork            = module.subnets.subnets["us-west1/${var.network_name}-subnet"].name
  backend_service       = google_compute_region_backend_service.this.self_link
  region                = "us-west1"
  load_balancing_scheme = "INTERNAL"
  all_ports             = true
}

module "routes" {
  source       = "../../modules/routes-beta"
  project_id   = var.project_id
  network_name = module.vpc.network_name
  routes_count = 2

  routes = [
    {
      name              = "${var.network_name}-egress-inet"
      description       = "route through IGW to access internet"
      destination_range = "0.0.0.0/0"
      tags              = "egress-inet"
      next_hop_internet = "true"
    },
    {
      name              = "${var.network_name}-ilb"
      description       = "route through ilb"
      destination_range = "10.10.20.0/24"
      next_hop_ilb      = google_compute_forwarding_rule.this.self_link
    },
  ]

  module_depends_on = [module.subnets.subnets, module.subnets-backup.subnets]
}
