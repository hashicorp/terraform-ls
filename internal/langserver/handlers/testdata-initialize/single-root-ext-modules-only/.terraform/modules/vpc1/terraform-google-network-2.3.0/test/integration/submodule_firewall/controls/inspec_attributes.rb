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

control "inspec_attributes" do
  title "Terraform Outputs"
  desc "Terraform Outputs"

  describe attribute("output_network_name") do
    it { should eq "#{network_name}" }
  end

  describe attribute("output_network_self_link") do
    it { should eq "https://www.googleapis.com/compute/v1/projects/#{project_id}/global/networks/#{network_name}" }
  end

  describe attribute("output_subnets_ips") do
    it { should eq ["10.10.10.0/24", "10.10.20.0/24"] }
  end

  describe attribute("output_routes") do
    it { should eq [] }
  end

  describe attribute("output_subnets_flow_logs") do
    it { should eq [false, true] }
  end

  describe attribute("output_subnets_names") do
    it { should eq ["#{network_name}-subnet-01", "#{network_name}-subnet-02"] }
  end

  describe attribute("output_subnets_private_access") do
    it { should eq [false, true] }
  end

  describe attribute("output_subnets_regions") do
    it { should eq ["us-west1", "us-west1"] }
  end

  describe attribute("output_subnets_secondary_ranges") do
    it { should eq [[],[]] }
  end

  describe attribute("output_project_id") do
    it { should eq project_id }
  end
end
