#!/bin/bash
# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


set -e

if [ -n "${GOOGLE_APPLICATION_CREDENTIALS}" ]; then
  export CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=${GOOGLE_APPLICATION_CREDENTIALS}
fi

PROJECT_ID=$1
NETWORK_ID=$2
FILTERED_ROUTES=$(gcloud compute routes list \
  --project="${PROJECT_ID}" \
  --format="value(name)" \
  --filter=" \
    nextHopGateway:(https://www.googleapis.com/compute/v1/projects/${PROJECT_ID}/global/gateways/default-internet-gateway) \
    AND network:(https://www.googleapis.com/compute/v1/projects/${PROJECT_ID}/global/networks/${NETWORK_ID}) \
    AND name~^default-route \
  "
)

function delete_internet_gateway_routes {
  local routes="${1}"
  echo "${routes}" | while read -r line; do
    echo "Deleting route ${line}..."
    gcloud compute routes delete "${line}" --quiet --project="${PROJECT_ID}"
  done
}

if [ -n "${FILTERED_ROUTES}" ]; then
  delete_internet_gateway_routes "${FILTERED_ROUTES}"
else
  echo "Default internet gateway route(s) not found; exiting..."
fi

