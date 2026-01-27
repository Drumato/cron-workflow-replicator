# Kustomize Integration Example

This example demonstrates how the CronWorkflow Replicator tool can automatically manage Kustomize `kustomization.yaml` files when generating CronWorkflow manifests.

## What This Example Does

This configuration:
1. Generates two CronWorkflow YAML files (`backup-job.yaml` and `cleanup-job.yaml`)
2. Automatically creates/updates a `kustomization.yaml` file that includes both generated files in its resources list
3. Preserves any existing resources in the kustomization.yaml file

## Configuration Features

The `config.yaml` demonstrates:
- **Kustomize Integration**: `kustomize.update-resources: true` enables automatic kustomization.yaml management
- **Multiple Workflows**: Two different CronWorkflows with different schedules and purposes
- **Path Resolution**: Output directory is relative to the config file location

## Generated Files

After running this example, you'll find:

### Generated CronWorkflows
- `output/backup-job.yaml` - Daily backup job (runs at 2 AM)
- `output/cleanup-job.yaml` - Daily cleanup job (runs at 4 AM)

### Auto-managed Kustomization
- `output/kustomization.yaml` - Automatically updated to include all generated resources

## How Kustomize Integration Works

1. **Initial State**: If `kustomization.yaml` doesn't exist, it's created with the generated files
2. **Updates**: If it exists, new generated files are added to the resources list
3. **Preservation**: Existing resources in the kustomization.yaml are preserved
4. **Error Handling**: If kustomization update fails, a warning is logged but processing continues

## Running This Example

```bash
# From the repository root
./cron-workflow-replicator --config examples/v1alpha1/kustomize/config.yaml

# Or using Docker
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/kustomize/config.yaml
```

## Use Cases

This feature is particularly useful when:
- You want to use `kubectl apply -k .` to deploy generated CronWorkflows
- You're managing multiple sets of CronWorkflows with Kustomize
- You have existing Kustomize resources that should be deployed alongside generated CronWorkflows
- You want to leverage Kustomize's patching and transformation capabilities with generated manifests

## Example kustomization.yaml Output

```yaml
APIVersion: kustomize.config.k8s.io/v1beta1
Kind: Kustomization
Resources:
- existing-resource.yaml  # Preserved from previous version
- backup-job.yaml         # Added by tool
- cleanup-job.yaml        # Added by tool
```