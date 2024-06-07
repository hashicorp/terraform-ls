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

project_id      = attribute('project_id')
network_01_name = attribute('network_01_name')
network_02_name = attribute('network_02_name')

control "gcloud" do
  title "gcloud configuration"

  describe command("gcloud compute routes describe '#{network_01_name}-egress-inet' --project=#{project_id} --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }
    let(:default_internet_gateway) { "https://www.googleapis.com/compute/v1/projects/#{project_id}/global/gateways/default-internet-gateway" }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    describe "destRange" do
      it "should equal '0.0.0.0/0'" do
        expect(data["destRange"]).to eq '0.0.0.0/0'
      end
    end

    describe "tags" do
      it "should equal 'egress-inet'" do
        expect(data["tags"]).to eq ['egress-inet']
      end
    end

    describe "nextHopGateway" do
      it "should equal the default internet gateway" do
        expect(data["nextHopGateway"]).to eq default_internet_gateway
      end
    end
  end

  describe command("gcloud compute routes describe '#{network_02_name}-egress-inet' --project=#{project_id} --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }
    let(:default_internet_gateway) { "https://www.googleapis.com/compute/v1/projects/#{project_id}/global/gateways/default-internet-gateway" }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    describe "destRange" do
      it "should equal '0.0.0.0/0'" do
        expect(data["destRange"]).to eq '0.0.0.0/0'
      end
    end

    describe "tags" do
      it "should equal 'egress-inet'" do
        expect(data["tags"]).to eq ['egress-inet']
      end
    end

    describe "nextHopGateway" do
      it "should equal the default internet gateway" do
        expect(data["nextHopGateway"]).to eq default_internet_gateway
      end
    end
  end

  describe command("gcloud compute routes describe '#{network_02_name}-testapp-proxy' --project=#{project_id} --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    describe "destRange" do
      it "should equal '10.50.10.0/24'" do
        expect(data["destRange"]).to eq '10.50.10.0/24'
      end
    end

    describe "tags" do
      it "should equal 'app-proxy'" do
        expect(data["tags"]).to eq ['app-proxy']
      end
    end

    describe "nextHopIp" do
      it "should equal '10.10.40.10'" do
        expect(data["nextHopIp"]).to eq '10.10.40.10'
      end
    end
  end
end
