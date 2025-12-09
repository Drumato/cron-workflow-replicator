## Development cycle

### When you change code

1. make sure all tests pass: `make test`
2. build the binary: `make build`
3. run the binary locally against local files to verify changes: `./bin/cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml`
4. commit your changes with a descriptive message

### When you add a new feature or fix a bug

1. add `examples/<version>/<feature-or-bug>/config.yaml` to demonstrate the feature or bug fix
3. run `NAME=<example> make example` to generate expected output files for the example
4. run `make test` to make sure all tests pass
5. commit your changes with a descriptive message
