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
forwarding_rule = attribute('forwarding_rule')

control "gcloud" do
  title "gcloud configuration"

  describe command("gcloud compute networks subnets describe #{network_name}-subnet --project=#{project_id} --region=us-west1 --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    it "purpose should be correct" do
      expect(data).to include(
        "purpose" => "PRIVATE",
      )
    end
    it "role should not exist" do
      expect(data).to_not include(
        "role"
      )
    end
  end

  describe command("gcloud compute networks subnets describe #{network_name}-subnet-01 --project=#{project_id} --region=us-west1 --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    it "purpose and role should be correct" do
      expect(data).to include(
        "purpose" => "INTERNAL_HTTPS_LOAD_BALANCER",
        "role" => "ACTIVE"
      )
    end
  end

  describe  command("gcloud compute networks subnets describe #{network_name}-subnet-02 --project=#{project_id} --region=us-west1 --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    it "purpose and role should be correct" do
      expect(data).to include(
        "purpose" => "INTERNAL_HTTPS_LOAD_BALANCER",
        "role" => "BACKUP"
      )
    end
  end

  describe command("gcloud compute routes describe '#{network_name}-ilb' --project=#{project_id} --format=json") do
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
      it "should equal '10.10.20.0/24'" do
        expect(data["destRange"]).to eq '10.10.20.0/24'
      end
    end

    describe "tags" do
      it "should equal 'egress-inet'" do
        expect(data["tags"]).to eq nil
      end
    end

    describe "nextHopIlb" do
      it "should equal the forwarding rule" do
        expect(data["nextHopIlb"]).to eq forwarding_rule
      end
    end
  end
end
