package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/drumato/cron-workflow-replicator/config"
	"github.com/drumato/cron-workflow-replicator/filesystem"
	"github.com/drumato/cron-workflow-replicator/kustomize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/types"
	kyaml "sigs.k8s.io/yaml"
)

func TestRunner_APIVersionConfiguration(t *testing.T) {
	// Create a temporary directory for test outputs
	tempDir := t.TempDir()

	tests := []struct {
		name               string
		unit               config.Unit
		expectedAPIVersion string
		expectedKind       string
	}{
		{
			name: "v1alpha1 API version",
			unit: config.Unit{
				OutputDirectory: "output", // Relative path
				APIVersion:      config.APIVersionV1Alpha1,
				Values: []config.Value{
					{
						Filename: "test-v1alpha1",
						Paths: []config.PathValue{
							{Path: "$.metadata.name", Value: "test-workflow"},
							{Path: "$.metadata.namespace", Value: "default"},
						},
					},
				},
			},
			expectedAPIVersion: "argoproj.io/v1alpha1",
			expectedKind:       "CronWorkflow",
		},
		{
			name: "empty API version defaults to v1alpha1",
			unit: config.Unit{
				OutputDirectory: "output", // Relative path
				APIVersion:      "",       // empty should default to v1alpha1
				Values: []config.Value{
					{
						Filename: "test-default",
						Paths: []config.PathValue{
							{Path: "$.metadata.name", Value: "test-workflow-default"},
							{Path: "$.metadata.namespace", Value: "default"},
						},
					},
				},
			},
			expectedAPIVersion: "argoproj.io/v1alpha1",
			expectedKind:       "CronWorkflow",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	runner := New(logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runner.processUnit(context.Background(), tt.unit, tempDir)
			assert.NoError(t, err)

			// Check that the output file was created
			outputFile := filepath.Join(tempDir, tt.unit.OutputDirectory, tt.unit.Values[0].Filename+".yaml")
			_, err = os.Stat(outputFile)
			assert.False(t, os.IsNotExist(err), "Output file %s was not created", outputFile)

			// Read and verify the content
			content, err := os.ReadFile(outputFile)
			assert.NoError(t, err)

			// Parse the YAML to verify APIVersion and Kind
			var result struct {
				APIVersion string `yaml:"apiVersion"`
				Kind       string `yaml:"kind"`
			}

			err = kyaml.Unmarshal(content, &result)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedAPIVersion, result.APIVersion)
			assert.Equal(t, tt.expectedKind, result.Kind)

			// Clean up the test file
			err = os.Remove(outputFile)
			assert.NoError(t, err)
		})
	}
}

func TestRunner_Run(t *testing.T) {
	// Create test base manifest content
	baseManifest := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: base-cronworkflow
  namespace: default
  labels:
    app: base-app
    environment: test
spec:
  schedule: "0 0 * * *"
  timezone: "Asia/Tokyo"
  suspend: false
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine:latest
        command: ["echo"]
        args: ["base workflow"]`

	tests := []struct {
		name                 string
		config               config.Config
		expectedFiles        []string
		expectedValues       map[string]func(*testing.T, []byte) // filename -> validation function
		baseManifestPath     string
		shouldCreateBaseFile bool
	}{
		{
			name: "novalue case",
			config: config.Config{
				Units: []config.Unit{
					{
						OutputDirectory: "./output",
						APIVersion:      config.APIVersionV1Alpha1,
						Values: []config.Value{
							{
								Filename: "novalue1",
								Paths:    []config.PathValue{},
							},
							{
								Filename: "novalue2",
								Paths:    []config.PathValue{},
							},
						},
					},
				},
			},
			expectedFiles: []string{
				"output/novalue1.yaml",
				"output/novalue2.yaml",
			},
			expectedValues: map[string]func(*testing.T, []byte){
				"output/novalue1.yaml": func(t *testing.T, content []byte) {
					validateBasicCronWorkflow(t, content, "", "")
				},
				"output/novalue2.yaml": func(t *testing.T, content []byte) {
					validateBasicCronWorkflow(t, content, "", "")
				},
			},
			shouldCreateBaseFile: false,
		},
		{
			name: "withvalue case",
			config: config.Config{
				Units: []config.Unit{
					{
						OutputDirectory: "./output",
						APIVersion:      config.APIVersionV1Alpha1,
						Values: []config.Value{
							{
								Filename: "withvalue1",
								Paths: []config.PathValue{
									{Path: "$.metadata.name", Value: "test-cronworkflow"},
									{Path: "$.metadata.namespace", Value: "test-namespace"},
									{Path: "$.metadata.labels.app", Value: "test-app"},
									{Path: "$.spec.schedule", Value: "0 0 * * *"},
									{Path: "$.spec.workflowSpec.entrypoint", Value: "main"},
								},
							},
							{
								Filename: "withvalue2",
								Paths: []config.PathValue{
									{Path: "$.metadata.name", Value: "another-cronworkflow"},
									{Path: "$.metadata.namespace", Value: "another-namespace"},
									{Path: "$.spec.schedule", Value: "0 12 * * 1"},
									{Path: "$.spec.workflowSpec.entrypoint", Value: "weekly-job"},
								},
							},
						},
					},
				},
			},
			expectedFiles: []string{
				"output/withvalue1.yaml",
				"output/withvalue2.yaml",
			},
			expectedValues: map[string]func(*testing.T, []byte){
				"output/withvalue1.yaml": func(t *testing.T, content []byte) {
					validateCronWorkflowWithValues(t, content, "test-cronworkflow", "test-namespace", "0 0 * * *", "main")
					validateLabels(t, content, map[string]string{"app": "test-app"})
				},
				"output/withvalue2.yaml": func(t *testing.T, content []byte) {
					validateCronWorkflowWithValues(t, content, "another-cronworkflow", "another-namespace", "0 12 * * 1", "weekly-job")
				},
			},
			shouldCreateBaseFile: false,
		},
		{
			name: "with base manifest path",
			config: config.Config{
				Units: []config.Unit{
					{
						BaseManifestPath: func() *string { s := "testdata/base-manifest.yaml"; return &s }(),
						OutputDirectory:  "./output",
						APIVersion:       config.APIVersionV1Alpha1,
						Values: []config.Value{
							{
								Filename: "customized1",
								Paths: []config.PathValue{
									{Path: "$.metadata.name", Value: "custom-cronworkflow"},
									{Path: "$.metadata.namespace", Value: "custom-namespace"},
									{Path: "$.metadata.labels.custom", Value: "label"},
									{Path: "$.spec.schedule", Value: "0 6 * * *"},
								},
							},
						},
					},
				},
			},
			expectedFiles: []string{
				"output/customized1.yaml",
			},
			expectedValues: map[string]func(*testing.T, []byte){
				"output/customized1.yaml": func(t *testing.T, content []byte) {
					cw := validateCronWorkflowContent(t, content)

					// Validate merged metadata
					assertCronWorkflowFields(t, cw, "custom-cronworkflow", "custom-namespace", "0 6 * * *")

					// Validate merged labels (base + custom)
					baseLabels := map[string]string{"app": "base-app", "environment": "test"}
					customLabels := map[string]string{"custom": "label"}
					assertMergedFields(t, cw, baseLabels, customLabels)

					// Validate merged spec (overrides + base values)
					assertCronWorkflowSpec(t, cw, "main", "Asia/Tokyo", &[]bool{false}[0])
				},
			},
			baseManifestPath:     "testdata/base-manifest.yaml",
			shouldCreateBaseFile: true,
		},
		{
			name: "without base manifest path (standalone)",
			config: config.Config{
				Units: []config.Unit{
					{
						BaseManifestPath: nil, // No base manifest
						OutputDirectory:  "./output",
						APIVersion:       config.APIVersionV1Alpha1,
						Values: []config.Value{
							{
								Filename: "standalone1",
								Paths: []config.PathValue{
									{Path: "$.metadata.name", Value: "standalone-cronworkflow"},
									{Path: "$.metadata.namespace", Value: "standalone-namespace"},
									{Path: "$.spec.schedule", Value: "0 12 * * 1"},
									{Path: "$.spec.workflowSpec.entrypoint", Value: "standalone-entrypoint"},
								},
							},
						},
					},
				},
			},
			expectedFiles: []string{
				"output/standalone1.yaml",
			},
			expectedValues: map[string]func(*testing.T, []byte){
				"output/standalone1.yaml": func(t *testing.T, content []byte) {
					cw := validateCronWorkflowContent(t, content)

					// Should only have the specified values, no base values
					assertCronWorkflowFields(t, cw, "standalone-cronworkflow", "standalone-namespace", "0 12 * * 1")

					// Timezone should be empty since no base manifest
					assert.Equal(t, "", cw.Spec.Timezone, "Timezone should be empty without base manifest")

					// Should have no labels from base manifest
					assert.Empty(t, cw.Labels, "Labels should be empty without base manifest")
				},
			},
			shouldCreateBaseFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use in-memory filesystem for testing
			fs := filesystem.NewInMemoryFileSystem()

			// Create base manifest file in filesystem if needed
			if tt.shouldCreateBaseFile && tt.baseManifestPath != "" {
				// Ensure directory exists
				dir := filepath.Dir(tt.baseManifestPath)
				err := fs.MkdirAll(dir, 0755)
				assert.NoError(t, err)

				// Create base manifest file
				file, err := fs.OpenFile(tt.baseManifestPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				assert.NoError(t, err)
				_, err = file.Write([]byte(baseManifest))
				assert.NoError(t, err)
				err = file.Close()
				assert.NoError(t, err)
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			fileReader := &FilesystemFileReader{fs: fs}
			runner := New(logger, WithFileSystem(fs), WithFileReader(fileReader))

			// Run the test
			ctx := context.Background()
			configDir := "."
			err := runner.Run(ctx, tt.config, configDir)
			assert.NoError(t, err)

			// Verify expected files were created
			for _, expectedFile := range tt.expectedFiles {
				file, err := fs.OpenFile(expectedFile, 0, 0)
				assert.NoError(t, err, "Expected file %s was not created", expectedFile)

				// Get file content
				memFile := file.(*filesystem.InMemoryFile)
				content := memFile.GetData()

				// Run validation function if provided
				if validateFunc, exists := tt.expectedValues[expectedFile]; exists {
					validateFunc(t, content)
				}

				err = file.Close()
				assert.NoError(t, err)
			}
		})
	}
}

// Enhanced helper functions for comprehensive validation
func validateCronWorkflowContent(t *testing.T, content []byte) *argoworkflowsv1alpha1.CronWorkflow {
	t.Helper()
	var cw argoworkflowsv1alpha1.CronWorkflow
	err := kyaml.Unmarshal(content, &cw)
	require.NoError(t, err, "Failed to unmarshal CronWorkflow YAML")

	// Validate basic required fields
	assert.Equal(t, "argoproj.io/v1alpha1", cw.APIVersion, "Incorrect APIVersion")
	assert.Equal(t, "CronWorkflow", cw.Kind, "Incorrect Kind")

	return &cw
}

func assertCronWorkflowFields(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow, expectedName, expectedNamespace, expectedSchedule string) {
	t.Helper()
	if expectedName != "" {
		assert.Equal(t, expectedName, cw.Name, "Incorrect CronWorkflow name")
	}
	if expectedNamespace != "" {
		assert.Equal(t, expectedNamespace, cw.Namespace, "Incorrect CronWorkflow namespace")
	}
	if expectedSchedule != "" {
		assert.Equal(t, expectedSchedule, cw.Spec.Schedule, "Incorrect CronWorkflow schedule")
	}
}

func assertCronWorkflowLabels(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow, expectedLabels map[string]string) {
	t.Helper()
	for key, expectedValue := range expectedLabels {
		actualValue, exists := cw.Labels[key]
		assert.True(t, exists, "Expected label '%s' not found", key)
		assert.Equal(t, expectedValue, actualValue, "Incorrect value for label '%s'", key)
	}
}

func assertCronWorkflowSpec(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow, expectedEntrypoint, expectedTimezone string, expectedSuspend *bool) {
	t.Helper()
	if expectedEntrypoint != "" {
		assert.Equal(t, expectedEntrypoint, cw.Spec.WorkflowSpec.Entrypoint, "Incorrect WorkflowSpec entrypoint")
	}
	if expectedTimezone != "" {
		assert.Equal(t, expectedTimezone, cw.Spec.Timezone, "Incorrect timezone")
	}
	if expectedSuspend != nil {
		assert.Equal(t, *expectedSuspend, cw.Spec.Suspend, "Incorrect suspend value")
	}
}

func assertMergedFields(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow, baseLabels, overrideLabels map[string]string) {
	t.Helper()
	// Check that base labels are preserved
	for key, expectedValue := range baseLabels {
		actualValue, exists := cw.Labels[key]
		assert.True(t, exists, "Base label '%s' should be preserved", key)
		assert.Equal(t, expectedValue, actualValue, "Base label '%s' value incorrect", key)
	}

	// Check that override labels are applied
	for key, expectedValue := range overrideLabels {
		actualValue, exists := cw.Labels[key]
		assert.True(t, exists, "Override label '%s' should be present", key)
		assert.Equal(t, expectedValue, actualValue, "Override label '%s' value incorrect", key)
	}
}

// Legacy helper functions for backward compatibility
func validateBasicCronWorkflow(t *testing.T, content []byte, expectedName, expectedNamespace string) {
	t.Helper()
	cw := validateCronWorkflowContent(t, content)
	assertCronWorkflowFields(t, cw, expectedName, expectedNamespace, "")
}

func validateCronWorkflowWithValues(t *testing.T, content []byte, expectedName, expectedNamespace, expectedSchedule, expectedEntrypoint string) {
	t.Helper()
	cw := validateCronWorkflowContent(t, content)
	assertCronWorkflowFields(t, cw, expectedName, expectedNamespace, expectedSchedule)
	assertCronWorkflowSpec(t, cw, expectedEntrypoint, "", nil)
}

func validateLabels(t *testing.T, content []byte, expectedLabels map[string]string) {
	t.Helper()
	cw := validateCronWorkflowContent(t, content)
	assertCronWorkflowLabels(t, cw, expectedLabels)
}

// FilesystemFileReader adapts filesystem.FileSystem to config.FileReader interface
type FilesystemFileReader struct {
	fs filesystem.FileSystem
}

func (f *FilesystemFileReader) ReadFile(filename string) ([]byte, error) {
	file, err := f.fs.OpenFile(filename, 0, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close() // In test context, close errors are typically not actionable for read-only files
	}()

	// Type assert to InMemoryFile to access GetData()
	if memFile, ok := file.(*filesystem.InMemoryFile); ok {
		return memFile.GetData(), nil
	}

	// For other filesystem types, we would need different handling
	return nil, fmt.Errorf("unsupported file type for reading")
}

func TestRunner_KustomizeIntegration(t *testing.T) {
	tests := []struct {
		name                      string
		unit                      config.Unit
		configDir                 string
		expectedFiles             []string
		expectedKustomization     *types.Kustomization
		shouldCreateKustomization bool
	}{
		{
			name: "kustomize enabled with multiple files",
			unit: config.Unit{
				OutputDirectory: "output",
				APIVersion:      config.APIVersionV1Alpha1,
				Kustomize: &config.KustomizeConfig{
					UpdateResources: true,
				},
				Values: []config.Value{
					{
						Filename: "backup-job",
					},
					{
						Filename: "cleanup-job",
					},
				},
			},
			configDir: "/config",
			expectedFiles: []string{
				"/config/output/backup-job.yaml",
				"/config/output/cleanup-job.yaml",
				"/config/output/kustomization.yaml",
			},
			expectedKustomization: &types.Kustomization{
				TypeMeta: types.TypeMeta{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
				},
				Resources: []string{"backup-job.yaml", "cleanup-job.yaml"},
			},
			shouldCreateKustomization: true,
		},
		{
			name: "kustomize disabled",
			unit: config.Unit{
				OutputDirectory: "output",
				APIVersion:      config.APIVersionV1Alpha1,
				Kustomize:       nil, // Not configured
				Values: []config.Value{
					{
						Filename: "backup-job",
					},
				},
			},
			configDir: "/config",
			expectedFiles: []string{
				"/config/output/backup-job.yaml",
			},
			shouldCreateKustomization: false,
		},
		{
			name: "kustomize configured but update-resources false",
			unit: config.Unit{
				OutputDirectory: "output",
				APIVersion:      config.APIVersionV1Alpha1,
				Kustomize: &config.KustomizeConfig{
					UpdateResources: false,
				},
				Values: []config.Value{
					{
						Filename: "backup-job",
					},
				},
			},
			configDir: "/config",
			expectedFiles: []string{
				"/config/output/backup-job.yaml",
			},
			shouldCreateKustomization: false,
		},
		{
			name: "existing kustomization.yaml gets updated",
			unit: config.Unit{
				OutputDirectory: "output",
				APIVersion:      config.APIVersionV1Alpha1,
				Kustomize: &config.KustomizeConfig{
					UpdateResources: true,
				},
				Values: []config.Value{
					{
						Filename: "backup-job",
					},
				},
			},
			configDir: "/config",
			expectedFiles: []string{
				"/config/output/backup-job.yaml",
				"/config/output/kustomization.yaml",
			},
			expectedKustomization: &types.Kustomization{
				TypeMeta: types.TypeMeta{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
				},
				Resources: []string{"existing-resource.yaml", "backup-job.yaml"},
			},
			shouldCreateKustomization: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use in-memory filesystem for testing
			fs := filesystem.NewMemoryFileSystem()
			kustomizeManager := kustomize.NewManager(fs)

			// Create existing kustomization.yaml for the "existing" test case
			if tt.name == "existing kustomization.yaml gets updated" {
				existingKustomization := types.Kustomization{
					TypeMeta: types.TypeMeta{
						APIVersion: "kustomize.config.k8s.io/v1beta1",
						Kind:       "Kustomization",
					},
					Resources: []string{"existing-resource.yaml"},
				}
				existingData, err := kyaml.Marshal(existingKustomization)
				require.NoError(t, err)

				outputDir := filepath.Join(tt.configDir, tt.unit.OutputDirectory)
				err = fs.MkdirAll(outputDir, 0755)
				require.NoError(t, err)
				err = fs.WriteFile(filepath.Join(outputDir, "kustomization.yaml"), existingData, 0644)
				require.NoError(t, err)
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			fileReader := &FilesystemFileReader{fs: fs}
			runner := New(logger,
				WithFileSystem(fs),
				WithFileReader(fileReader),
				WithKustomizeManager(kustomizeManager))

			// Run the test
			ctx := context.Background()
			err := runner.processUnit(ctx, tt.unit, tt.configDir)
			assert.NoError(t, err)

			// Verify expected files were created
			for _, expectedFile := range tt.expectedFiles {
				assert.True(t, fs.Exists(expectedFile), "Expected file %s was not created", expectedFile)
			}

			// Verify kustomization.yaml content if it should be created
			if tt.shouldCreateKustomization {
				kustomizationPath := filepath.Join(tt.configDir, tt.unit.OutputDirectory, "kustomization.yaml")
				assert.True(t, fs.Exists(kustomizationPath), "kustomization.yaml should have been created")

				data, err := fs.ReadFile(kustomizationPath)
				require.NoError(t, err)

				var actualKustomization types.Kustomization
				err = kyaml.Unmarshal(data, &actualKustomization)
				require.NoError(t, err)

				assert.Equal(t, tt.expectedKustomization.APIVersion, actualKustomization.APIVersion)
				assert.Equal(t, tt.expectedKustomization.Kind, actualKustomization.Kind)
				assert.ElementsMatch(t, tt.expectedKustomization.Resources, actualKustomization.Resources)
			} else {
				// Verify kustomization.yaml was not created
				kustomizationPath := filepath.Join(tt.configDir, tt.unit.OutputDirectory, "kustomization.yaml")
				assert.False(t, fs.Exists(kustomizationPath), "kustomization.yaml should not have been created")
			}
		})
	}
}

// ErrorFileReader simulates various file reading errors
type ErrorFileReader struct {
	files  map[string][]byte
	errors map[string]error
}

func NewErrorFileReader() *ErrorFileReader {
	return &ErrorFileReader{
		files:  make(map[string][]byte),
		errors: make(map[string]error),
	}
}

func (r *ErrorFileReader) AddFile(path string, content []byte) {
	r.files[path] = content
}

func (r *ErrorFileReader) AddError(path string, err error) {
	r.errors[path] = err
}

func (r *ErrorFileReader) ReadFile(filename string) ([]byte, error) {
	if err, exists := r.errors[filename]; exists {
		return nil, err
	}
	if content, exists := r.files[filename]; exists {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func TestRunner_processUnit_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name        string
		setupFS     func(*filesystem.InMemoryFileSystem)
		setupReader func(*ErrorFileReader)
		unit        config.Unit
		configDir   string
		expectedErr string
	}{
		{
			name: "base manifest read error",
			setupFS: func(fs *filesystem.InMemoryFileSystem) {
				// FS setup not needed for this error case
			},
			setupReader: func(reader *ErrorFileReader) {
				reader.AddError("/config/base.yaml", errors.New("permission denied"))
			},
			unit: config.Unit{
				BaseManifestPath: func() *string { s := "base.yaml"; return &s }(),
				OutputDirectory:  "output",
				APIVersion:       config.APIVersionV1Alpha1,
				Values: []config.Value{
					{
						Filename: "test-workflow",
					},
				},
			},
			configDir:   "/config",
			expectedErr: "failed to load base CronWorkflow: failed to read base manifest file /config/base.yaml: permission denied",
		},
		{
			name: "invalid YAML in base manifest",
			setupFS: func(fs *filesystem.InMemoryFileSystem) {
				// FS setup not needed for this error case
			},
			setupReader: func(reader *ErrorFileReader) {
				invalidYAML := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: invalid
  invalid-syntax: [
    - broken yaml`
				reader.AddFile("/config/bad.yaml", []byte(invalidYAML))
			},
			unit: config.Unit{
				BaseManifestPath: func() *string { s := "bad.yaml"; return &s }(),
				OutputDirectory:  "output",
				APIVersion:       config.APIVersionV1Alpha1,
				Values: []config.Value{
					{
						Filename: "test-workflow",
					},
				},
			},
			configDir:   "/config",
			expectedErr: "failed to load base CronWorkflow: failed to unmarshal base manifest file /config/bad.yaml",
		},
		{
			name: "output directory creation failure",
			setupFS: func(fs *filesystem.InMemoryFileSystem) {
				// Create a file where we want to create a directory
				err := fs.WriteFile("/config/output", []byte("blocking file"), 0644)
				require.NoError(t, err)
			},
			setupReader: func(reader *ErrorFileReader) {
				// No base manifest needed for this test
			},
			unit: config.Unit{
				OutputDirectory: "output",
				APIVersion:      config.APIVersionV1Alpha1,
				Values: []config.Value{
					{
						Filename: "test-workflow",
					},
				},
			},
			configDir:   "/config",
			expectedErr: "failed to create output directory /config/output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup filesystem
			fs := filesystem.NewInMemoryFileSystem()
			if tt.setupFS != nil {
				tt.setupFS(fs)
			}

			// Setup error file reader
			reader := NewErrorFileReader()
			if tt.setupReader != nil {
				tt.setupReader(reader)
			}

			// Create runner
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			kustomizeManager := kustomize.NewManager(fs)
			runner := New(logger,
				WithFileSystem(fs),
				WithFileReader(reader),
				WithKustomizeManager(kustomizeManager))

			// Run the test
			ctx := context.Background()
			err := runner.processUnit(ctx, tt.unit, tt.configDir)

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestRunner_Run_ErrorPropagation(t *testing.T) {
	// Test that errors in individual units propagate properly to the Run method
	fs := filesystem.NewInMemoryFileSystem()
	// Create a file where we want to create a directory to simulate filesystem error
	err := fs.WriteFile("/config/output", []byte("blocking file"), 0644)
	require.NoError(t, err)

	reader := NewErrorFileReader()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	kustomizeManager := kustomize.NewManager(fs)
	runner := New(logger,
		WithFileSystem(fs),
		WithFileReader(reader),
		WithKustomizeManager(kustomizeManager))

	cfg := config.Config{
		Units: []config.Unit{
			{
				OutputDirectory: "output",
				APIVersion:      config.APIVersionV1Alpha1,
				Values: []config.Value{
					{
						Filename: "test-workflow",
					},
				},
			},
		},
	}

	ctx := context.Background()
	err = runner.Run(ctx, cfg, "/config")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create output directory")
}

func TestRunner_MemoryUsage_InMemoryGeneration(t *testing.T) {
	// Test that in-memory generation doesn't leak memory
	fs := filesystem.NewInMemoryFileSystem()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	fileReader := &FilesystemFileReader{fs: fs}
	kustomizeManager := kustomize.NewManager(fs)
	runner := New(logger, WithFileSystem(fs), WithFileReader(fileReader), WithKustomizeManager(kustomizeManager))

	// Create a config with multiple units and files to stress test memory usage
	cfg := config.Config{
		Units: []config.Unit{},
	}

	// Generate many units to test memory usage
	for i := 0; i < 100; i++ {
		unit := config.Unit{
			OutputDirectory: fmt.Sprintf("output-%d", i),
			APIVersion:      config.APIVersionV1Alpha1,
			Values:          []config.Value{},
		}

		// Generate many values per unit
		for j := 0; j < 10; j++ {
			unit.Values = append(unit.Values, config.Value{
				Filename: fmt.Sprintf("workflow-%d-%d", i, j),
			})
		}

		cfg.Units = append(cfg.Units, unit)
	}

	ctx := context.Background()
	err := runner.Run(ctx, cfg, "/config")
	assert.NoError(t, err, "Should handle large number of files without error")

	// Verify that files were created (spot check)
	assert.True(t, fs.Exists("/config/output-0/workflow-0-0.yaml"))
	assert.True(t, fs.Exists("/config/output-50/workflow-50-5.yaml"))
	assert.True(t, fs.Exists("/config/output-99/workflow-99-9.yaml"))

	// Test doesn't verify specific memory usage, but ensures no crashes or errors
	// In a real environment, you could use runtime.ReadMemStats() to check memory usage
}

func BenchmarkRunner_InMemoryGeneration(b *testing.B) {
	// Benchmark in-memory file generation vs theoretical file I/O performance
	baseManifest := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: base-cronworkflow
  namespace: default
spec:
  schedule: "0 0 * * *"
  workflowSpec:
    entrypoint: main`

	fs := filesystem.NewInMemoryFileSystem()
	// Create base manifest
	err := fs.MkdirAll("testdata", 0755)
	require.NoError(b, err)
	err = fs.WriteFile("testdata/base-manifest.yaml", []byte(baseManifest), 0644)
	require.NoError(b, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	fileReader := &FilesystemFileReader{fs: fs}
	runner := New(logger, WithFileSystem(fs), WithFileReader(fileReader))

	cfg := config.Config{
		Units: []config.Unit{
			{
				BaseManifestPath: func() *string { s := "testdata/base-manifest.yaml"; return &s }(),
				OutputDirectory:  "output",
				APIVersion:       config.APIVersionV1Alpha1,
				Values: []config.Value{
					{
						Filename: "benchmark-workflow",
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear previous run output
		if fs.Exists("/config/output") {
			// Simple approach: recreate the filesystem for each benchmark iteration
			fs = filesystem.NewInMemoryFileSystem()
			err := fs.MkdirAll("testdata", 0755)
			require.NoError(b, err)
			err = fs.WriteFile("testdata/base-manifest.yaml", []byte(baseManifest), 0644)
			require.NoError(b, err)
			fileReader = &FilesystemFileReader{fs: fs}
			runner = New(logger, WithFileSystem(fs), WithFileReader(fileReader))
		}

		ctx := context.Background()
		err := runner.Run(ctx, cfg, "/config")
		require.NoError(b, err)
	}
}

func TestRunner_ConcurrentInMemoryAccess(t *testing.T) {
	// Test concurrent access to in-memory filesystem doesn't cause race conditions
	// Note: Each goroutine uses its own filesystem to avoid concurrent access issues
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create multiple runners to simulate concurrent usage
	numGoroutines := 10
	errors := make(chan error, numGoroutines)
	filesystems := make([]*filesystem.InMemoryFileSystem, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			// Each goroutine gets its own filesystem to avoid race conditions
			fs := filesystem.NewInMemoryFileSystem()
			fileReader := &FilesystemFileReader{fs: fs}
			runner := New(logger, WithFileSystem(fs), WithFileReader(fileReader))
			filesystems[goroutineID] = fs

			cfg := config.Config{
				Units: []config.Unit{
					{
						OutputDirectory: "output",
						APIVersion:      config.APIVersionV1Alpha1,
						Values: []config.Value{
							{
								Filename: fmt.Sprintf("concurrent-workflow-%d", goroutineID),
							},
						},
					},
				},
			}

			ctx := context.Background()
			err := runner.Run(ctx, cfg, "/config")
			errors <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		assert.NoError(t, err, "Concurrent access should not cause errors")
	}

	// Verify all files were created in their respective filesystems
	for i := 0; i < numGoroutines; i++ {
		if filesystems[i] != nil {
			expectedFile := fmt.Sprintf("/config/output/concurrent-workflow-%d.yaml", i)
			assert.True(t, filesystems[i].Exists(expectedFile), "File %s should exist in filesystem %d", expectedFile, i)
		}
	}
}

func TestRunner_ContentValidation_ComprehensiveFieldChecking(t *testing.T) {
	// Comprehensive test that validates ALL generated content structure
	baseManifest := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: comprehensive-base
  namespace: base-ns
  labels:
    base-label: base-value
    shared-label: base-shared
  annotations:
    base-annotation: base-annotation-value
spec:
  schedule: "0 0 * * *"
  timezone: "Asia/Tokyo"
  suspend: false
  startingDeadlineSeconds: 300
  concurrencyPolicy: "Forbid"
  workflowSpec:
    entrypoint: base-entrypoint
    arguments:
      parameters:
      - name: base-param
        value: base-param-value`

	fs := filesystem.NewInMemoryFileSystem()
	// Create base manifest
	err := fs.MkdirAll("/config/testdata", 0755)
	require.NoError(t, err)
	err = fs.WriteFile("/config/testdata/comprehensive-base.yaml", []byte(baseManifest), 0644)
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	fileReader := &FilesystemFileReader{fs: fs}
	kustomizeManager := kustomize.NewManager(fs)
	runner := New(logger, WithFileSystem(fs), WithFileReader(fileReader), WithKustomizeManager(kustomizeManager))

	cfg := config.Config{
		Units: []config.Unit{
			{
				BaseManifestPath: func() *string { s := "/config/testdata/comprehensive-base.yaml"; return &s }(),
				OutputDirectory:  "output",
				APIVersion:       config.APIVersionV1Alpha1,
				Values: []config.Value{
					{
						Filename: "comprehensive-test",
						Paths: []config.PathValue{
							{Path: "$.metadata.name", Value: "comprehensive-cronworkflow"},
							{Path: "$.metadata.namespace", Value: "override-ns"},
							{Path: "$.metadata.labels.override-label", Value: "override-value"},
							{Path: "$.metadata.labels.shared-label", Value: "override-shared"},
							{Path: "$.metadata.annotations.override-annotation", Value: "override-annotation-value"},
							{Path: "$.spec.schedule", Value: "0 6 * * *"},
							{Path: "$.spec.suspend", Value: "true"},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	err = runner.Run(ctx, cfg, "/config")
	require.NoError(t, err)

	// Read and comprehensively validate the generated file
	file, err := fs.OpenFile("/config/output/comprehensive-test.yaml", 0, 0)
	require.NoError(t, err)
	memFile := file.(*filesystem.InMemoryFile)
	content := memFile.GetData()
	err = file.Close()
	require.NoError(t, err)

	// Use comprehensive validation
	cw := validateCronWorkflowContent(t, content)

	// Validate overridden fields
	assertCronWorkflowFields(t, cw, "comprehensive-cronworkflow", "override-ns", "0 6 * * *")

	// Validate merged labels (base + override)
	expectedLabels := map[string]string{
		"base-label":     "base-value",      // From base
		"shared-label":   "override-shared", // Overridden
		"override-label": "override-value",  // New
	}
	assertCronWorkflowLabels(t, cw, expectedLabels)

	// Validate merged annotations
	assert.Equal(t, "base-annotation-value", cw.Annotations["base-annotation"], "Base annotation should be preserved")
	assert.Equal(t, "override-annotation-value", cw.Annotations["override-annotation"], "Override annotation should be present")

	// Validate merged spec fields
	assert.Equal(t, "0 6 * * *", cw.Spec.Schedule, "Schedule should be overridden")
	assert.Equal(t, true, cw.Spec.Suspend, "Suspend should be overridden")
	assert.Equal(t, "Asia/Tokyo", cw.Spec.Timezone, "Timezone should be preserved from base")
	assert.Equal(t, argoworkflowsv1alpha1.ConcurrencyPolicy("Forbid"), cw.Spec.ConcurrencyPolicy, "ConcurrencyPolicy should be preserved from base")
	if assert.NotNil(t, cw.Spec.StartingDeadlineSeconds, "StartingDeadlineSeconds should be preserved") {
		assert.Equal(t, int64(300), *cw.Spec.StartingDeadlineSeconds, "StartingDeadlineSeconds value should be correct")
	}

	// Validate WorkflowSpec fields
	assert.Equal(t, "base-entrypoint", cw.Spec.WorkflowSpec.Entrypoint, "WorkflowSpec.Entrypoint should be preserved from base")
	assert.NotNil(t, cw.Spec.WorkflowSpec.Arguments, "WorkflowSpec.Arguments should be preserved")
	assert.Len(t, cw.Spec.WorkflowSpec.Arguments.Parameters, 1, "Should have base parameters")
	assert.Equal(t, "base-param", cw.Spec.WorkflowSpec.Arguments.Parameters[0].Name, "Base parameter name should be preserved")
}
