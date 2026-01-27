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

## Examples

Check the `examples/` directory for complete configuration examples:

- `examples/v1alpha1/novalue/` - Basic configuration without custom values
- `examples/v1alpha1/withvalue/` - Configuration with custom values
- `examples/v1alpha1/basemanifest/` - Configuration using base manifest templates
- `examples/v1alpha1/kustomize/` - Configuration with Kustomize integration enabled