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

  describe command("gcloud compute firewall-rules describe #{network_name}-ingress-internal --project=#{project_id} --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    describe "internal rule" do
      it "should exist" do
        expect(data).to include(
          "sourceRanges" => ["10.10.20.0/24", "10.10.10.0/24"]
        )
      end
    end

    describe "allowed internal rules" do
      it "should contain ICMP rule" do
        expect(data["allowed"]).to include({"IPProtocol" => "icmp"})
      end

      it "should contain UDP rule" do
        expect(data["allowed"]).to include({"IPProtocol" => "udp"})
      end

      it "should contain TCP rule" do
        expect(data["allowed"]).to include({"IPProtocol"=>"tcp", "ports"=>["8080", "1000-2000"]})
      end
    end
  end

  # Custom rules
  describe command("gcloud compute firewall-rules describe allow-backend-to-databases --project=#{project_id} --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    describe "Custom TAG rule" do
      it "has backend tag as source" do
        expect(data).to include(
          "sourceTags" => ["backed"]
        )
      end

      it "has databases tag as target" do
        expect(data).to include(
          "targetTags" => ["databases"]
        )
      end

      it "has expected TCP rule" do
        expect(data["allowed"]).to include(
            {
              "IPProtocol" => "tcp",
              "ports" => ["3306", "5432", "1521", "1433"]
            }
        )
      end
    end
  end

describe command("gcloud compute firewall-rules describe deny-ingress-6534-6566 --project=#{project_id} --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    describe "deny-ingress-6534-6566" do
      it "should be disabled" do
        expect(data).to include(
          "disabled" => true
        )
      end

      it "has 0.0.0.0/0 source range" do
        expect(data).to include(
          "sourceRanges" => ["0.0.0.0/0"]
        )
      end

      it "has expected TCP rules" do
        expect(data["denied"]).to include(
            {
              "IPProtocol" => "tcp",
              "ports" => ["6534-6566"]
            }
        )
      end

      it "has expected UDP rules" do
        expect(data["denied"]).to include(
            {
              "IPProtocol" => "udp",
              "ports" => ["6534-6566"]
            }
        )
      end
    end
  end


describe command("gcloud compute firewall-rules describe allow-all-admin-sa --project=#{project_id} --format=json") do
    its(:exit_status) { should eq 0 }
    its(:stderr) { should eq '' }

    let(:data) do
      if subject.exit_status == 0
        JSON.parse(subject.stdout)
      else
        {}
      end
    end

    describe "allow-all-admin-sa" do
      it "should be enabled" do
        expect(data).to include(
          "disabled" => false
        )
      end

      it "should has correct source SA" do
        expect(data["sourceServiceAccounts"]).to eq(["admin@my-shiny-org.iam.gserviceaccount.com"])
      end

      it "should has priority 30" do
        expect(data["priority"]).to eq(30)
      end

      it "has expected TCP rules" do
        expect(data["allowed"]).to include(
            {
              "IPProtocol" => "tcp"
            }
        )
      end

      it "has expected UDP rules" do
        expect(data["allowed"]).to include(
            {
              "IPProtocol" => "udp"
            }
        )
      end
    end
  end

end

