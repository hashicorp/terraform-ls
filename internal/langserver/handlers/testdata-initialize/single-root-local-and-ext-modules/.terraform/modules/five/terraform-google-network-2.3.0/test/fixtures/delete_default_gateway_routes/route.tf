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

# This fixture defines a default internet gateway route that DOESN'T start
# with 'default-route' to test the behavior of the script that deletes
# the default internet gateway routes.

resource "google_compute_route" "alternative_gateway" {
  project = var.project_id
  network = module.example.network_name

  name             = "alternative-gateway-route"
  description      = "Alternative gateway route"
  dest_range       = "0.0.0.0/0"
  tags             = ["egress-inet"]
  next_hop_gateway = "default-internet-gateway"
}
