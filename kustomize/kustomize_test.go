package kustomize

import (
	"path/filepath"
	"strings"
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
	assert.Contains(t, err.Error(), "failed to parse existing kustomization.yaml")
}

func TestManager_UpdateKustomization_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		setupFS       func(*filesystem.InMemoryFileSystem) string
		outputDir     string
		generatedFiles []string
		expectedError string
	}{
		{
			name: "empty output directory",
			setupFS: func(fs *filesystem.InMemoryFileSystem) string {
				return ""
			},
			outputDir:     "",
			generatedFiles: []string{"test.yaml"},
			expectedError: "output directory cannot be empty",
		},
		{
			name: "empty generated files list",
			setupFS: func(fs *filesystem.InMemoryFileSystem) string {
				outputDir := "/test/output"
				err := fs.MkdirAll(outputDir, 0755)
				require.NoError(t, err)
				return outputDir
			},
			outputDir:     "/test/output",
			generatedFiles: []string{},
			expectedError: "", // Should not error, just return early
		},
		{
			name: "mixed valid and invalid files",
			setupFS: func(fs *filesystem.InMemoryFileSystem) string {
				outputDir := "/test/output"
				err := fs.MkdirAll(outputDir, 0755)
				require.NoError(t, err)
				return outputDir
			},
			outputDir:     "/test/output",
			generatedFiles: []string{"valid.yaml", "invalid.txt", "", "another.yml"},
			expectedError: "", // Should filter and process only valid files
		},
		{
			name: "all invalid files",
			setupFS: func(fs *filesystem.InMemoryFileSystem) string {
				outputDir := "/test/output"
				err := fs.MkdirAll(outputDir, 0755)
				require.NoError(t, err)
				return outputDir
			},
			outputDir:     "/test/output",
			generatedFiles: []string{"invalid.txt", "another.json", ""},
			expectedError: "", // Should return early with no valid files
		},
		{
			name: "empty kustomization file exists",
			setupFS: func(fs *filesystem.InMemoryFileSystem) string {
				outputDir := "/test/output"
				err := fs.MkdirAll(outputDir, 0755)
				require.NoError(t, err)

				// Create empty kustomization.yaml
				kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")
				err = fs.WriteFile(kustomizationPath, []byte(""), 0644)
				require.NoError(t, err)
				return outputDir
			},
			outputDir:     "/test/output",
			generatedFiles: []string{"test.yaml"},
			expectedError: "", // Should handle empty file gracefully
		},
		{
			name: "kustomization file with missing fields",
			setupFS: func(fs *filesystem.InMemoryFileSystem) string {
				outputDir := "/test/output"
				err := fs.MkdirAll(outputDir, 0755)
				require.NoError(t, err)

				// Create kustomization.yaml with missing required fields
				incompleteKustomization := `resources:
  - existing.yaml`
				kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")
				err = fs.WriteFile(kustomizationPath, []byte(incompleteKustomization), 0644)
				require.NoError(t, err)
				return outputDir
			},
			outputDir:     "/test/output",
			generatedFiles: []string{"new.yaml"},
			expectedError: "", // Should fix missing fields and continue
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := filesystem.NewInMemoryFileSystem()
			outputDir := tt.outputDir
			if tt.setupFS != nil {
				outputDir = tt.setupFS(fs)
				if outputDir != tt.outputDir && tt.outputDir != "" {
					outputDir = tt.outputDir // Use the specified outputDir if different
				}
			}

			manager := NewManager(fs)
			err := manager.UpdateKustomization(outputDir, tt.generatedFiles)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)

				// Verify kustomization.yaml was created/updated if valid files were provided
				validFilesCount := 0
				for _, file := range tt.generatedFiles {
					if file != "" && (strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml")) {
						validFilesCount++
					}
				}

				if validFilesCount > 0 {
					kustomizationPath := filepath.Join(outputDir, "kustomization.yaml")
					assert.True(t, fs.Exists(kustomizationPath), "kustomization.yaml should be created")

					// Verify the content is valid YAML
					data, readErr := fs.ReadFile(kustomizationPath)
					assert.NoError(t, readErr)

					var result Kustomization
					unmarshalErr := kyaml.Unmarshal(data, &result)
					assert.NoError(t, unmarshalErr)

					// Verify required fields are set
					assert.Equal(t, "kustomize.config.k8s.io/v1beta1", result.APIVersion)
					assert.Equal(t, "Kustomization", result.Kind)
					assert.NotNil(t, result.Resources)
				}
			}
		})
	}
}
