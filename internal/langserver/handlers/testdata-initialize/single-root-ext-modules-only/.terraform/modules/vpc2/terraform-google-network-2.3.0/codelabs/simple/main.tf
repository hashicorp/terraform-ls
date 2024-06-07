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

resource "random_id" "network_id" {
  byte_length = 8
}

resource "google_project_service" "compute" {
  service = "compute.googleapis.com"
}

# Create the network
module "vpc" {
  source  = "terraform-google-modules/network/google"
  version = "~> 0.4.0"

  # Give the network a name and project
  project_id   = google_project_service.compute.project
  network_name = "my-custom-vpc-${random_id.network_id.hex}"

  subnets = [
    {
      # Creates your first subnet in us-west1 and defines a range for it
      subnet_name   = "my-first-subnet"
      subnet_ip     = "10.10.10.0/24"
      subnet_region = "us-west1"
    },
    {
      # Creates a dedicated subnet for GKE
      subnet_name   = "my-gke-subnet"
      subnet_ip     = "10.10.20.0/24"
      subnet_region = "us-west1"
    },
  ]

  # Define secondary ranges for each of your subnets
  secondary_ranges = {
    my-first-subnet = []

    my-gke-subnet = [
      {
        # Define a secondary range for Kubernetes pods to use
        range_name    = "my-gke-pods-range"
        ip_cidr_range = "192.168.64.0/24"
      },
    ]
  }
}

resource "random_id" "instance_id" {
  byte_length = 8
}

# Launch a VM on it
resource "google_compute_instance" "default" {
  name         = "vm-${random_id.instance_id.hex}"
  project      = google_project_service.compute.project
  machine_type = "f1-micro"
  zone         = "us-west1-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-9"
    }
  }

  network_interface {
    subnetwork         = module.vpc.subnets_names[0]
    subnetwork_project = google_project_service.compute.project

    access_config {
      # Include this section to give the VM an external ip address
    }
  }

  # Apply the firewall rule to allow external IPs to ping this instance
  tags = ["allow-ping"]
}

# Allow traffic to the VM
resource "google_compute_firewall" "allow-ping" {
  name    = "default-ping"
  network = module.vpc.network_name
  project = google_project_service.compute.project

  allow {
    protocol = "icmp"
  }

  # Allow traffic from everywhere to instances with an http-server tag
  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["allow-ping"]
}

output "ip" {
  value = google_compute_instance.default.network_interface.0.access_config.0.nat_ip
}
