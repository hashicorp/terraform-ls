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

project_id = attribute('project_id')
peerings   = attribute('peerings')

control "gcloud" do
  title "gcloud configuration"
  peerings.each do |key, value|
    local_network_peering   = value['local_network_peering']
    peer_network_peering    = value['peer_network_peering']
    local_network_self_link = local_network_peering['network']
    peer_network_self_link  = peer_network_peering['network']
    local_network_name      = local_network_self_link.split('/')[-1]
    peer_network_name       = peer_network_self_link.split('/')[-1]

    describe command("gcloud compute networks peerings list --project=#{project_id} --network=#{local_network_name} --format=json") do
      its(:exit_status) { should eq 0 }
      its(:stderr) { should eq '' }

      let(:data) do
        if subject.exit_status == 0
          JSON.parse(subject.stdout)
        else
          {}
        end
      end

      describe "local VPC peering" do
        it "should exist" do
          expect(data[0]['peerings'].select{|x| x['name'] == local_network_peering['name']}).not_to be_empty
        end
        it "should be active" do
          expect(data[0]['peerings'].select{|x| x['name'] == local_network_peering['name']}[0]['state']).to eq(
            "ACTIVE"
          )
        end
        it "should be connected to #{peer_network_name} network" do
          expect(data[0]['peerings'].select{|x| x['name'] == local_network_peering['name']}[0]['network']).to eq(
            peer_network_self_link
          )
        end
        it "should export custom routes" do
          expect(data[0]['peerings'].select{|x| x['name'] == local_network_peering['name']}[0]['exportCustomRoutes']).to eq(
            true
          )
        end
        it "should not import custom routes" do
          expect(data[0]['peerings'].select{|x| x['name'] == local_network_peering['name']}[0]['importCustomRoutes']).to eq(
            false
          )
        end
      end

    end

    describe command("gcloud compute networks peerings list --project=#{project_id} --network=#{peer_network_name} --format=json") do
      its(:exit_status) { should eq 0 }
      its(:stderr) { should eq '' }

      let(:data) do
        if subject.exit_status == 0
          JSON.parse(subject.stdout)
        else
          {}
        end
      end

      describe "peer VPC peering" do
        it "should exist" do
          expect(data[0]['peerings'].select{|x| x['name'] == peer_network_peering['name']}).not_to be_empty
        end
        it "should be active" do
          expect(data[0]['peerings'].select{|x| x['name'] == peer_network_peering['name']}[0]['state']).to eq(
            "ACTIVE"
          )
        end
        it "should be connected to #{local_network_name} network" do
          expect(data[0]['peerings'].select{|x| x['name'] == peer_network_peering['name']}[0]['network']).to eq(
            local_network_self_link
          )
        end
        it "should not export custom routes" do
          expect(data[0]['peerings'].select{|x| x['name'] == peer_network_peering['name']}[0]['exportCustomRoutes']).to eq(
            false
          )
        end
        it "should import custom routes" do
          expect(data[0]['peerings'].select{|x| x['name'] == peer_network_peering['name']}[0]['importCustomRoutes']).to eq(
            true
          )
        end
      end
    end
  end
end
