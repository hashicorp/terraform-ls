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
  subnet_02 = "${var.network_name}-subnet-02"
}

module "test-vpc-module" {
  source       = "../../"
  project_id   = var.project_id
  network_name = var.network_name

  subnets = [
    {
      subnet_name   = local.subnet_01
      subnet_ip     = "10.10.10.0/24"
      subnet_region = "us-west1"
    },
    {
      subnet_name           = local.subnet_02
      subnet_ip             = "10.10.20.0/24"
      subnet_region         = "us-west1"
      subnet_private_access = "true"
      subnet_flow_logs      = "true"
    },
  ]
}

// Custom firewall rules
locals {
  custom_rules = {
    // Example of custom tcp/udp rule
    deny-ingress-6534-6566 = {
      description          = "Deny all INGRESS to port 6534-6566"
      direction            = "INGRESS"
      action               = "deny"
      ranges               = ["0.0.0.0/0"] # source or destination ranges (depends on `direction`)
      use_service_accounts = false         # if `true` targets/sources expect list of instances SA, if false - list of tags
      targets              = null          # target_service_accounts or target_tags depends on `use_service_accounts` value
      sources              = null          # source_service_accounts or source_tags depends on `use_service_accounts` value
      rules = [{
        protocol = "tcp"
        ports    = ["6534-6566"]
        },
        {
          protocol = "udp"
          ports    = ["6534-6566"]
      }]

      extra_attributes = {
        disabled = true
        priority = 95
      }
    }

    // Example how to allow connection from instances with `backend` tag, to instances with `databases` tag
    allow-backend-to-databases = {
      description          = "Allow backend nodes connection to databases instances"
      direction            = "INGRESS"
      action               = "allow"
      ranges               = null
      use_service_accounts = false
      targets              = ["databases"] # target_tags
      sources              = ["backed"]    # source_tags
      rules = [{
        protocol = "tcp"
        ports    = ["3306", "5432", "1521", "1433"]
      }]

      extra_attributes = {}
    }

    // Example how to allow connection from an instance with a given service account
    allow-all-admin-sa = {
      description          = "Allow all traffic from admin sa instances"
      direction            = "INGRESS"
      action               = "allow"
      ranges               = null
      use_service_accounts = true
      targets              = null
      sources              = ["admin@my-shiny-org.iam.gserviceaccount.com"]
      rules = [{
        protocol = "tcp"
        ports    = null # all ports
        },
        {
          protocol = "udp"
          ports    = null # all ports
        }
      ]
      extra_attributes = {
        priority = 30
      }
    }
  }
}



module "test-firewall-submodule" {
  source                  = "../../modules/fabric-net-firewall"
  project_id              = var.project_id
  network                 = module.test-vpc-module.network_name
  internal_ranges_enabled = true
  internal_ranges         = module.test-vpc-module.subnets_ips

  internal_allow = [
    {
      protocol = "icmp"
    },
    {
      protocol = "tcp",
      ports    = ["8080", "1000-2000"]
    },
    {
      protocol = "udp"
      # all ports will be opened if `ports` key isn't specified
    },
  ]
  custom_rules = local.custom_rules
}
