package runner

import (
	"context"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
						Metadata: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "default",
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
						Metadata: metav1.ObjectMeta{
							Name:      "test-workflow-default",
							Namespace: "default",
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
								Metadata: metav1.ObjectMeta{},
								Spec:     argoworkflowsv1alpha1.CronWorkflowSpec{},
							},
							{
								Filename: "novalue2",
								Metadata: metav1.ObjectMeta{},
								Spec:     argoworkflowsv1alpha1.CronWorkflowSpec{},
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
								Metadata: metav1.ObjectMeta{
									Name:      "test-cronworkflow",
									Namespace: "test-namespace",
									Labels: map[string]string{
										"app": "test-app",
									},
								},
								Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
									Schedule: "0 0 * * *",
									WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
										Entrypoint: "main",
									},
								},
							},
							{
								Filename: "withvalue2",
								Metadata: metav1.ObjectMeta{
									Name:      "another-cronworkflow",
									Namespace: "another-namespace",
								},
								Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
									Schedule: "0 12 * * 1",
									WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
										Entrypoint: "weekly-job",
									},
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
								Metadata: metav1.ObjectMeta{
									Name:      "custom-cronworkflow",
									Namespace: "custom-namespace",
									Labels: map[string]string{
										"custom": "label",
									},
								},
								Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
									Schedule: "0 6 * * *", // Override schedule
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
					var cw argoworkflowsv1alpha1.CronWorkflow
					err := kyaml.Unmarshal(content, &cw)
					assert.NoError(t, err)

					// Should have merged metadata
					assert.Equal(t, "custom-cronworkflow", cw.Name)
					assert.Equal(t, "custom-namespace", cw.Namespace)

					// Should have base labels plus custom labels
					assert.Equal(t, "base-app", cw.Labels["app"])
					assert.Equal(t, "test", cw.Labels["environment"])
					assert.Equal(t, "label", cw.Labels["custom"])

					// Should have overridden schedule but keep base timezone and other specs
					assert.Equal(t, "0 6 * * *", cw.Spec.Schedule)
					assert.Equal(t, "Asia/Tokyo", cw.Spec.Timezone)
					assert.Equal(t, "main", cw.Spec.WorkflowSpec.Entrypoint)
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
								Metadata: metav1.ObjectMeta{
									Name:      "standalone-cronworkflow",
									Namespace: "standalone-namespace",
								},
								Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
									Schedule: "0 12 * * 1",
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
					var cw argoworkflowsv1alpha1.CronWorkflow
					err := kyaml.Unmarshal(content, &cw)
					assert.NoError(t, err)

					// Should only have the specified values, no base values
					assert.Equal(t, "standalone-cronworkflow", cw.Name)
					assert.Equal(t, "0 12 * * 1", cw.Spec.Schedule)
					// Timezone should be empty since no base manifest
					assert.Equal(t, "", cw.Spec.Timezone)
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

// Helper functions for validation
func validateBasicCronWorkflow(t *testing.T, content []byte, expectedName, expectedNamespace string) {
	var cw argoworkflowsv1alpha1.CronWorkflow
	err := kyaml.Unmarshal(content, &cw)
	assert.NoError(t, err)

	assert.Equal(t, "argoproj.io/v1alpha1", cw.APIVersion)
	assert.Equal(t, "CronWorkflow", cw.Kind)

	if expectedName != "" {
		assert.Equal(t, expectedName, cw.Name)
	}
	if expectedNamespace != "" {
		assert.Equal(t, expectedNamespace, cw.Namespace)
	}
}

func validateCronWorkflowWithValues(t *testing.T, content []byte, expectedName, expectedNamespace, expectedSchedule, expectedEntrypoint string) {
	validateBasicCronWorkflow(t, content, expectedName, expectedNamespace)

	var cw argoworkflowsv1alpha1.CronWorkflow
	err := kyaml.Unmarshal(content, &cw)
	assert.NoError(t, err)

	assert.Equal(t, expectedSchedule, cw.Spec.Schedule)
	assert.Equal(t, expectedEntrypoint, cw.Spec.WorkflowSpec.Entrypoint)
}

func validateLabels(t *testing.T, content []byte, expectedLabels map[string]string) {
	var cw argoworkflowsv1alpha1.CronWorkflow
	err := kyaml.Unmarshal(content, &cw)
	assert.NoError(t, err)

	for key, expectedValue := range expectedLabels {
		actualValue, exists := cw.Labels[key]
		assert.True(t, exists, "Expected label '%s' not found", key)
		assert.Equal(t, expectedValue, actualValue)
	}
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
		expectedKustomization     *kustomize.Kustomization
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
						Metadata: metav1.ObjectMeta{
							Name:      "backup-cronworkflow",
							Namespace: "default",
						},
						Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
							Schedule: "0 2 * * *",
						},
					},
					{
						Filename: "cleanup-job",
						Metadata: metav1.ObjectMeta{
							Name:      "cleanup-cronworkflow",
							Namespace: "default",
						},
						Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
							Schedule: "0 4 * * *",
						},
					},
				},
			},
			configDir: "/config",
			expectedFiles: []string{
				"/config/output/backup-job.yaml",
				"/config/output/cleanup-job.yaml",
				"/config/output/kustomization.yaml",
			},
			expectedKustomization: &kustomize.Kustomization{
				APIVersion: "kustomize.config.k8s.io/v1beta1",
				Kind:       "Kustomization",
				Resources:  []string{"backup-job.yaml", "cleanup-job.yaml"},
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
						Metadata: metav1.ObjectMeta{
							Name:      "backup-cronworkflow",
							Namespace: "default",
						},
						Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
							Schedule: "0 2 * * *",
						},
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
						Metadata: metav1.ObjectMeta{
							Name:      "backup-cronworkflow",
							Namespace: "default",
						},
						Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
							Schedule: "0 2 * * *",
						},
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
						Metadata: metav1.ObjectMeta{
							Name:      "backup-cronworkflow",
							Namespace: "default",
						},
						Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
							Schedule: "0 2 * * *",
						},
					},
				},
			},
			configDir: "/config",
			expectedFiles: []string{
				"/config/output/backup-job.yaml",
				"/config/output/kustomization.yaml",
			},
			expectedKustomization: &kustomize.Kustomization{
				APIVersion: "kustomize.config.k8s.io/v1beta1",
				Kind:       "Kustomization",
				Resources:  []string{"existing-resource.yaml", "backup-job.yaml"},
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
				existingKustomization := kustomize.Kustomization{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
					Resources:  []string{"existing-resource.yaml"},
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

				var actualKustomization kustomize.Kustomization
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
