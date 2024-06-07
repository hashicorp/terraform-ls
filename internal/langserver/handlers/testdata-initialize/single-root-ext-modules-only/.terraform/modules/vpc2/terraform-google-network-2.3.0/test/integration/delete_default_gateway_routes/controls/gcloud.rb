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

control "gcloud" do
  title "gcloud configuration"

  # Verify that no routes whose names begin with 'default-route' and whose
  # nextHopGateway is the default-internet-gateway exist
  describe command("gcloud compute routes list --project=#{project_id} --filter=\"nextHopGateway:https://www.googleapis.com/compute/v1/projects/#{project_id}/global/gateways/default-internet-gateway AND network:https://www.googleapis.com/compute/v1/projects/#{project_id}/global/networks/#{network_name}\" --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    describe "routes" do
      it "should only be one" do
        expect(data.length).to eq 1
      end

      it "should not begin with 'default-route'" do
        expect(data.first["name"]).not_to match(/^default-route/)
      end
    end
  end
end
