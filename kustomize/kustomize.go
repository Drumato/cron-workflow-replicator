package kustomize

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/drumato/cron-workflow-replicator/filesystem"
	"sigs.k8s.io/kustomize/api/types"
	kyaml "sigs.k8s.io/yaml"
)

// We use the official Kustomization type from sigs.k8s.io/kustomize/api/types

// Manager handles kustomization.yaml file operations
type Manager struct {
	fs filesystem.FileSystem
}

// NewManager creates a new kustomize manager
func NewManager(fs filesystem.FileSystem) *Manager {
	return &Manager{
		fs: fs,
	}
}

// UpdateKustomization updates the kustomization.yaml file in the given output directory
// with the provided list of generated files
// If recreate is true, the existing kustomization.yaml will be completely recreated instead of merged
func (m *Manager) UpdateKustomization(outputDir string, generatedFiles []string, recreate bool) error {
	// Input validation
	if outputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}
	if len(generatedFiles) == 0 {
		slog.Debug("no generated files provided for kustomization update", "outputDir", outputDir)
		return nil // Nothing to do if no files were generated
	}

	// Validate generated files
	validFiles := make([]string, 0, len(generatedFiles))
	for _, file := range generatedFiles {
		if file == "" {
			slog.Warn("ignoring empty filename in generated files list", "outputDir", outputDir)
			continue
		}
		if !strings.HasSuffix(file, ".yaml") && !strings.HasSuffix(file, ".yml") {
			slog.Warn("ignoring non-YAML file in generated files list", "file", file, "outputDir", outputDir)
			continue
		}
		validFiles = append(validFiles, file)
	}

	if len(validFiles) == 0 {
		slog.Debug("no valid YAML files to add to kustomization", "outputDir", outputDir, "originalCount", len(generatedFiles))
		return nil
	}

	kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")

	// Initialize with default kustomization structure using official types
	kustomization := &types.Kustomization{
		TypeMeta: types.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		Resources: []string{},
	}

	// Try to read existing kustomization.yaml if it exists and recreate is false
	if !recreate && m.fs.Exists(kustomizationPath) {
		slog.Debug("reading existing kustomization.yaml", "path", kustomizationPath)

		data, err := m.fs.ReadFile(kustomizationPath)
		if err != nil {
			return fmt.Errorf("failed to read existing kustomization.yaml at %s: %w. "+
				"Check file permissions and ensure the directory is accessible", kustomizationPath, err)
		}

		if len(data) == 0 {
			slog.Warn("existing kustomization.yaml is empty, using default structure", "path", kustomizationPath)
		} else {
			if err := kyaml.Unmarshal(data, kustomization); err != nil {
				// Provide more helpful error message for YAML parsing errors
				return fmt.Errorf("failed to parse existing kustomization.yaml at %s: %w. "+
					"The file may contain invalid YAML syntax. Please verify the file format or delete it to recreate",
					kustomizationPath, err)
			}

			// Validate the parsed kustomization structure
			if kustomization.APIVersion == "" {
				kustomization.APIVersion = "kustomize.config.k8s.io/v1beta1"
				slog.Info("setting missing apiVersion in existing kustomization.yaml", "path", kustomizationPath)
			}
			if kustomization.Kind == "" {
				kustomization.Kind = "Kustomization"
				slog.Info("setting missing kind in existing kustomization.yaml", "path", kustomizationPath)
			}
			if kustomization.Resources == nil {
				kustomization.Resources = []string{}
				slog.Info("initializing empty resources list in existing kustomization.yaml", "path", kustomizationPath)
			}
		}
	} else if recreate {
		slog.Debug("recreating kustomization.yaml (recreate mode)", "path", kustomizationPath)
	} else {
		slog.Debug("creating new kustomization.yaml", "path", kustomizationPath)
	}

	// Add new resources, avoiding duplicates
	existingResources := make(map[string]bool)
	for _, resource := range kustomization.Resources {
		existingResources[resource] = true
	}

	newResourcesAdded := 0
	for _, file := range validFiles {
		// Just use the filename (not full path) since files are in the same directory as kustomization.yaml
		filename := filepath.Base(file)
		if !existingResources[filename] {
			kustomization.Resources = append(kustomization.Resources, filename)
			newResourcesAdded++
		} else {
			slog.Debug("resource already exists in kustomization.yaml", "resource", filename, "path", kustomizationPath)
		}
	}

	slog.Debug("kustomization update summary",
		"path", kustomizationPath,
		"totalResources", len(kustomization.Resources),
		"newResourcesAdded", newResourcesAdded,
		"validFilesProvided", len(validFiles))

	// Marshal and write the updated kustomization.yaml
	data, err := kyaml.Marshal(kustomization)
	if err != nil {
		return fmt.Errorf("failed to marshal kustomization data to YAML: %w. "+
			"This may indicate an internal error with the kustomization structure", err)
	}

	if err := m.fs.WriteFile(kustomizationPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write kustomization.yaml to %s: %w. "+
			"Check directory permissions and available disk space. "+
			"Ensure the output directory is writable", kustomizationPath, err)
	}

	if recreate {
		slog.Info("successfully recreated kustomization.yaml",
			"path", kustomizationPath,
			"totalResources", len(kustomization.Resources),
			"mode", "recreate")
	} else {
		slog.Info("successfully updated kustomization.yaml",
			"path", kustomizationPath,
			"totalResources", len(kustomization.Resources),
			"newResourcesAdded", newResourcesAdded,
			"mode", "merge")
	}

	return nil
}
