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

# vpc 1
output "network_01_name" {
  value       = module.test-vpc-module-01.network_name
  description = "The name of the VPC network-01"
}

output "network_01_self_link" {
  value       = module.test-vpc-module-01.network_self_link
  description = "The URI of the VPC network-01"
}

output "network_01_subnets" {
  value       = module.test-vpc-module-01.subnets_names
  description = "The names of the subnets being created on network-01"
}

output "network_01_subnets_ips" {
  value       = module.test-vpc-module-01.subnets_ips
  description = "The IP and cidrs of the subnets being created on network-01"
}

output "network_01_subnets_regions" {
  value       = module.test-vpc-module-01.subnets_regions
  description = "The region where the subnets will be created on network-01"
}

output "network_01_subnets_private_access" {
  value       = module.test-vpc-module-01.subnets_private_access
  description = "Whether the subnets will have access to Google API's without a public IP on network-01"
}

output "network_01_subnets_flow_logs" {
  value       = module.test-vpc-module-01.subnets_flow_logs
  description = "Whether the subnets will have VPC flow logs enabled"
}

output "network_01_subnets_secondary_ranges" {
  value       = module.test-vpc-module-01.subnets_secondary_ranges
  description = "The secondary ranges associated with these subnets on network-01"
}

output "network_01_routes" {
  value       = module.test-vpc-module-01.route_names
  description = "The routes associated with network-01"
}

# vpc 2
output "network_02_name" {
  value       = module.test-vpc-module-02.network_name
  description = "The name of the VPC network-02"
}

output "network_02_self_link" {
  value       = module.test-vpc-module-02.network_self_link
  description = "The URI of the VPC network-02"
}

output "network_02_subnets" {
  value       = module.test-vpc-module-02.subnets_names
  description = "The names of the subnets being created on network-02"
}

output "network_02_subnets_ips" {
  value       = module.test-vpc-module-02.subnets_ips
  description = "The IP and cidrs of the subnets being created on network-02"
}

output "network_02_subnets_regions" {
  value       = module.test-vpc-module-02.subnets_regions
  description = "The region where the subnets will be created on network-02"
}

output "network_02_subnets_private_access" {
  value       = module.test-vpc-module-02.subnets_private_access
  description = "Whether the subnets will have access to Google API's without a public IP on network-02"
}

output "network_02_subnets_flow_logs" {
  value       = module.test-vpc-module-02.subnets_flow_logs
  description = "Whether the subnets will have VPC flow logs enabled"
}

output "network_02_subnets_secondary_ranges" {
  value       = module.test-vpc-module-02.subnets_secondary_ranges
  description = "The secondary ranges associated with these subnets on network-02"
}

output "network_02_routes" {
  value       = module.test-vpc-module-02.route_names
  description = "The routes associated with network-02"
}
