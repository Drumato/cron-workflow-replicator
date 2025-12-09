# CronWorkflow Replicator

this is a simple tool to replicate Kubernetes CronWorkflows with different values.

## Roadmap

- [x] replicates CronWorkflow manifests with different values
- [x] supports multiple input files
- [x] can read baseManifestPath that is specified in config file
  - if the baseManifestPath is specified, the replicator reads the base manifest and unmarshal into base CronWorkflow object.
  - then, it applies the values from the config file to the base CronWorkflow object.
