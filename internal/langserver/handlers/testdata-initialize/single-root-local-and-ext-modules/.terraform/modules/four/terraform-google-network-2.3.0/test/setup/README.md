# Integration Testing

Use this directory to create resources reflecting the same resource fixtures
created for use by the CI environment CI integration test pipelines.  The intent
of these resources is to run the integration tests locally as closely as
possible to how they will run in the CI system.

Once created, store the service account key content into the
`SERVICE_ACCOUNT_JSON` environment variable. This reflects the same behavior
as used in CI.

For example:

```bash
terraform init
terraform apply
mkdir -p ~/.credentials
terraform output sa_key | base64 --decode > ~/.credentials/network-sa.json
```

Then, configure the environment (suggest using direnv) like so:

```bash
export SERVICE_ACCOUNT_JSON=$(cat ${HOME}/.credentials/network-sa.json)
export PROJECT_ID="network-module"
```

With these variables set, change to the root of the module and execute the
`make test_integration` task. This make target is the same that is executed
by this module's CI pipeline during integration testing, and will run the
integration tests from your machine.

Alternatively, to run the integration tests directly from the Docker
container used by the module's CI pipeline, perform the above steps and then
run the `make test_integration_docker` target
