package kustomize

import (
	"path/filepath"
	"testing"

	"github.com/drumato/cron-workflow-replicator/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kyaml "sigs.k8s.io/yaml"
)

func TestManager_UpdateKustomization_NewFile(t *testing.T) {
	fs := filesystem.NewMemoryFileSystem()
	manager := NewManager(fs)

	outputDir := "/output"
	generatedFiles := []string{"backup-job.yaml", "cleanup-job.yaml"}

	err := manager.UpdateKustomization(outputDir, generatedFiles)
	require.NoError(t, err)

	// Check that kustomization.yaml was created
	kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")
	assert.True(t, fs.Exists(kustomizationPath))

	// Read and verify the content
	data, err := fs.ReadFile(kustomizationPath)
	require.NoError(t, err)

	var kustomization Kustomization
	err = kyaml.Unmarshal(data, &kustomization)
	require.NoError(t, err)

	assert.Equal(t, "kustomize.config.k8s.io/v1beta1", kustomization.APIVersion)
	assert.Equal(t, "Kustomization", kustomization.Kind)
	assert.Contains(t, kustomization.Resources, "backup-job.yaml")
	assert.Contains(t, kustomization.Resources, "cleanup-job.yaml")
	assert.Len(t, kustomization.Resources, 2)
}

func TestManager_UpdateKustomization_ExistingFile(t *testing.T) {
	fs := filesystem.NewMemoryFileSystem()
	manager := NewManager(fs)

	outputDir := "/output"
	kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")

	// Create existing kustomization.yaml
	existingKustomization := Kustomization{
		APIVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		Resources:  []string{"existing-job.yaml"},
	}
	existingData, err := kyaml.Marshal(existingKustomization)
	require.NoError(t, err)

	err = fs.MkdirAll(outputDir, 0755)
	require.NoError(t, err)
	err = fs.WriteFile(kustomizationPath, existingData, 0644)
	require.NoError(t, err)

	// Update with new files
	generatedFiles := []string{"backup-job.yaml", "cleanup-job.yaml"}
	err = manager.UpdateKustomization(outputDir, generatedFiles)
	require.NoError(t, err)

	// Read and verify the updated content
	data, err := fs.ReadFile(kustomizationPath)
	require.NoError(t, err)

	var kustomization Kustomization
	err = kyaml.Unmarshal(data, &kustomization)
	require.NoError(t, err)

	assert.Equal(t, "kustomize.config.k8s.io/v1beta1", kustomization.APIVersion)
	assert.Equal(t, "Kustomization", kustomization.Kind)
	assert.Contains(t, kustomization.Resources, "existing-job.yaml")
	assert.Contains(t, kustomization.Resources, "backup-job.yaml")
	assert.Contains(t, kustomization.Resources, "cleanup-job.yaml")
	assert.Len(t, kustomization.Resources, 3)
}

func TestManager_UpdateKustomization_NoDuplicates(t *testing.T) {
	fs := filesystem.NewMemoryFileSystem()
	manager := NewManager(fs)

	outputDir := "/output"
	kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")

	// Create existing kustomization.yaml with some resources
	existingKustomization := Kustomization{
		APIVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		Resources:  []string{"backup-job.yaml", "existing-job.yaml"},
	}
	existingData, err := kyaml.Marshal(existingKustomization)
	require.NoError(t, err)

	err = fs.MkdirAll(outputDir, 0755)
	require.NoError(t, err)
	err = fs.WriteFile(kustomizationPath, existingData, 0644)
	require.NoError(t, err)

	// Try to add files, including one that already exists
	generatedFiles := []string{"backup-job.yaml", "cleanup-job.yaml"}
	err = manager.UpdateKustomization(outputDir, generatedFiles)
	require.NoError(t, err)

	// Read and verify the updated content
	data, err := fs.ReadFile(kustomizationPath)
	require.NoError(t, err)

	var kustomization Kustomization
	err = kyaml.Unmarshal(data, &kustomization)
	require.NoError(t, err)

	// Should not have duplicates
	assert.Contains(t, kustomization.Resources, "backup-job.yaml")
	assert.Contains(t, kustomization.Resources, "existing-job.yaml")
	assert.Contains(t, kustomization.Resources, "cleanup-job.yaml")
	assert.Len(t, kustomization.Resources, 3) // No duplicates
}

func TestManager_UpdateKustomization_EmptyFilesList(t *testing.T) {
	fs := filesystem.NewMemoryFileSystem()
	manager := NewManager(fs)

	outputDir := "/output"
	generatedFiles := []string{}

	err := manager.UpdateKustomization(outputDir, generatedFiles)
	require.NoError(t, err)

	// Should not create kustomization.yaml if no files to add
	kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")
	assert.False(t, fs.Exists(kustomizationPath))
}

func TestManager_UpdateKustomization_FilenamesOnly(t *testing.T) {
	fs := filesystem.NewMemoryFileSystem()
	manager := NewManager(fs)

	outputDir := "/output"
	// Pass full paths, but only filenames should be stored
	generatedFiles := []string{"/some/path/backup-job.yaml", "cleanup-job.yaml"}

	err := manager.UpdateKustomization(outputDir, generatedFiles)
	require.NoError(t, err)

	// Read and verify the content
	kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")
	data, err := fs.ReadFile(kustomizationPath)
	require.NoError(t, err)

	var kustomization Kustomization
	err = kyaml.Unmarshal(data, &kustomization)
	require.NoError(t, err)

	// Should only contain filenames, not full paths
	assert.Contains(t, kustomization.Resources, "backup-job.yaml")
	assert.Contains(t, kustomization.Resources, "cleanup-job.yaml")
	assert.NotContains(t, kustomization.Resources, "/some/path/backup-job.yaml")
	assert.Len(t, kustomization.Resources, 2)
}

func TestManager_UpdateKustomization_InvalidExistingFile(t *testing.T) {
	fs := filesystem.NewMemoryFileSystem()
	manager := NewManager(fs)

	outputDir := "/output"
	kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")

	// Create invalid YAML content
	err := fs.MkdirAll(outputDir, 0755)
	require.NoError(t, err)
	err = fs.WriteFile(kustomizationPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	generatedFiles := []string{"backup-job.yaml"}
	err = manager.UpdateKustomization(outputDir, generatedFiles)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal kustomization.yaml")
}
