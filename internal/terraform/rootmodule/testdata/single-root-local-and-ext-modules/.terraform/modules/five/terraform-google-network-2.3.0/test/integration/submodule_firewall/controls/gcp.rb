# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

project_id   = attribute('project_id')
network_name = attribute('network_name')

control "gcp" do
  title "Google Cloud configuration"

  describe google_compute_firewalls(project: project_id) do
    its('firewall_names') { should include "#{network_name}-ingress-internal" }
    its('firewall_names') { should include "#{network_name}-ingress-tag-http" }
    its('firewall_names') { should include "#{network_name}-ingress-tag-https" }
    its('firewall_names') { should include "#{network_name}-ingress-tag-ssh" }
    its('firewall_names') { should_not include "default-ingress-admins" }
    its('firewall_names') { should include "deny-ingress-6534-6566" }
    its('firewall_names') { should include "allow-backend-to-databases" }
    its('firewall_names') { should include "allow-all-admin-sa" }
  end

end
