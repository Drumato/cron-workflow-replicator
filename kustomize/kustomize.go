package kustomize

import (
	"fmt"
	"path/filepath"

	"github.com/drumato/cron-workflow-replicator/filesystem"
	kyaml "sigs.k8s.io/yaml"
)

// Kustomization represents the structure of a kustomization.yaml file
type Kustomization struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Resources  []string `yaml:"resources"`
}

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
func (m *Manager) UpdateKustomization(outputDir string, generatedFiles []string) error {
	if len(generatedFiles) == 0 {
		return nil // Nothing to do if no files were generated
	}

	kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")

	// Try to read existing kustomization.yaml
	kustomization := &Kustomization{
		APIVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		Resources:  []string{},
	}

	if m.fs.Exists(kustomizationPath) {
		data, err := m.fs.ReadFile(kustomizationPath)
		if err != nil {
			return fmt.Errorf("failed to read existing kustomization.yaml: %w", err)
		}

		if err := kyaml.Unmarshal(data, kustomization); err != nil {
			return fmt.Errorf("failed to unmarshal kustomization.yaml: %w", err)
		}
	}

	// Add new resources, avoiding duplicates
	existingResources := make(map[string]bool)
	for _, resource := range kustomization.Resources {
		existingResources[resource] = true
	}

	for _, file := range generatedFiles {
		// Just use the filename (not full path) since files are in the same directory as kustomization.yaml
		filename := filepath.Base(file)
		if !existingResources[filename] {
			kustomization.Resources = append(kustomization.Resources, filename)
		}
	}

	// Marshal and write the updated kustomization.yaml
	data, err := kyaml.Marshal(kustomization)
	if err != nil {
		return fmt.Errorf("failed to marshal kustomization.yaml: %w", err)
	}

	if err := m.fs.WriteFile(kustomizationPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write kustomization.yaml: %w", err)
	}

	return nil
}
