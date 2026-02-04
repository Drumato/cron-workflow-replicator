# Configuration

This document explains how to configure the CronWorkflow Replicator tool.

## Path Resolution

The tool resolves relative paths in configuration files relative to the config file location, not the current working directory. This ensures consistent behavior regardless of where you run the command.

### Supported Path Fields

- `outputDirectory`: Output directory for generated YAML files
- `baseManifestPath`: Path to the base CronWorkflow manifest template

### Path Resolution Behavior

The tool handles path resolution in specific ways:

- **Relative paths**: Always resolved from the configuration file's directory, not your current working directory
- **Absolute paths**: Used as-is without any modification
- **Nested paths**: Work correctly for both relative and absolute paths

### Path Examples

```yaml
units:
  - outputDirectory: "./output"              # Resolved relative to config file
    baseManifestPath: "./base-manifest.yaml" # Resolved relative to config file
    # ...
  - outputDirectory: "manifests/output"      # Nested relative path
    baseManifestPath: "templates/base.yaml"  # Nested relative path
    # ...
  - outputDirectory: "/absolute/path/output"         # Absolute paths work as-is
    baseManifestPath: "/absolute/path/base.yaml"     # Absolute paths work as-is
    # ...
```

### Running from Different Directories

This means you can run the tool from any directory and it will work correctly:

```bash
# These all work the same way:
./cron-workflow-replicator --config examples/v1alpha1/basemanifest/config.yaml
cd /tmp && /path/to/cron-workflow-replicator --config /path/to/examples/v1alpha1/basemanifest/config.yaml
```

### Path Resolution Edge Cases

- If your config file is in `/project/configs/app.yaml` and specifies `outputDirectory: "./output"`, files will be written to `/project/configs/output/`, not `./output` relative to where you run the command
- This behavior applies consistently to both `outputDirectory` and `baseManifestPath` fields

## Kustomize Integration

The tool can automatically manage Kustomize kustomization.yaml files when generating CronWorkflow manifests.

### Enabling Kustomize Integration

Add the kustomize configuration to your unit:

```yaml
units:
  - outputDirectory: "./output"
    baseManifestPath: "./base-manifest.yaml"
    kustomize:
      update-resources: true
    # ... rest of configuration
```

### How Kustomize Integration Works

When `kustomize.update-resources: true` is set:

1. The tool generates CronWorkflow YAML files in the specified output directory
2. It automatically creates or updates a `kustomization.yaml` file in the same directory
3. The `kustomization.yaml` includes all generated files in its `resources` list
4. If kustomization update fails, it logs a warning but continues processing other files

### Example

If your unit generates files like `workflow-1.yaml`, `workflow-2.yaml`, the tool will create:

```yaml
# kustomization.yaml (auto-generated)
resources:
- workflow-1.yaml
- workflow-2.yaml
```

## File Naming and Collision Handling

When multiple values in your configuration would generate files with the same name, the tool automatically handles collisions by adding numeric suffixes.

### Naming Behavior

- First file: `filename.yaml`
- Second file with same name: `filename-2.yaml`
- Third file with same name: `filename-3.yaml`
- And so on...

### Example

If your configuration generates multiple workflows that would all be named `daily-job.yaml`, you'll get:
- `daily-job.yaml`
- `daily-job-2.yaml`
- `daily-job-3.yaml`

This ensures no files are overwritten and all generated manifests are preserved.

## Configuration File Structure

The configuration file defines how CronWorkflows should be generated. Each `unit` in the configuration represents a set of CronWorkflows to be created.

### Basic Configuration

```yaml
units:
  - outputDirectory: "./output"
    # Basic unit configuration
```

### With Base Manifest

```yaml
units:
  - outputDirectory: "./output"
    baseManifestPath: "./base-manifest.yaml"
    # Unit configuration with base template
```

### With Custom Values

```yaml
units:
  - outputDirectory: "./output"
    baseManifestPath: "./base-manifest.yaml"
    # Custom values can be injected into templates
```

## Value Configuration with JSONPath

### New JSONPath-Based Configuration

Starting with the latest version, the tool uses JSONPath expressions to set values in generated CronWorkflows. This provides more flexibility and precision when configuring generated manifests.

### Basic JSONPath Structure

```yaml
units:
  - outputDirectory: "./output"
    baseManifestPath: "./base-manifest.yaml"
    values:
      - filename: "example-workflow"
        paths:
          - path: "$.metadata.name"
            value: "my-cronworkflow"
          - path: "$.metadata.namespace"
            value: "default"
          - path: "$.spec.schedule"
            value: "0 0 * * *"
```

### JSONPath Expression Rules

- All paths must start with `$` (root element)
- Use dot notation for nested fields: `$.metadata.name`
- Array indexing supported: `$.spec.workflowSpec.templates[0].name`
- Validation ensures path is valid JSONPath expression
- Empty `paths` array is allowed (useful for templates without customization)

### Common JSONPath Examples

```yaml
# Setting metadata fields
- path: "$.metadata.name"
  value: "my-cronworkflow"
- path: "$.metadata.namespace"
  value: "production"
- path: "$.metadata.labels.app"
  value: "data-processor"

# Setting spec fields
- path: "$.spec.schedule"
  value: "0 2 * * *"
- path: "$.spec.concurrencyPolicy"
  value: "Forbid"

# Setting nested workflow spec fields
- path: "$.spec.workflowSpec.entrypoint"
  value: "main"
- path: "$.spec.workflowSpec.templates[0].name"
  value: "worker-task"

# Setting arguments and parameters
- path: "$.spec.workflowSpec.arguments.parameters[0].name"
  value: "input-file"
- path: "$.spec.workflowSpec.arguments.parameters[0].value"
  value: "/data/input.csv"
```

### Migration from Old Format

**Old format (no longer supported):**
```yaml
# OLD - No longer works
values:
  - filename: "example"
    metadata:
      name: "my-cronworkflow"
    spec:
      schedule: "0 0 * * *"
```

**New format:**
```yaml
# NEW - Current format
values:
  - filename: "example"
    paths:
      - path: "$.metadata.name"
        value: "my-cronworkflow"
      - path: "$.spec.schedule"
        value: "0 0 * * *"
```

### JSONPath Benefits

- **Precision**: Target exact fields without affecting other parts of the manifest
- **Flexibility**: Set any field in the generated YAML, including deeply nested ones
- **Validation**: JSONPath expressions are validated at parse time
- **Clarity**: Explicit path declarations make configurations self-documenting

## Examples

Check the `examples/` directory for complete configuration examples:

- `examples/v1alpha1/novalue/` - Basic configuration without custom values
- `examples/v1alpha1/withvalue/` - Configuration with custom values
- `examples/v1alpha1/basemanifest/` - Configuration using base manifest templates
- `examples/v1alpha1/kustomize/` - Configuration with Kustomize integration enabled