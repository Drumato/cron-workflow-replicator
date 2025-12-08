package runner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/drumato/cron-workflow-replicator/config"
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
				OutputDirectory: tempDir,
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
				OutputDirectory: tempDir,
				APIVersion:      "", // empty should default to v1alpha1
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
			err := runner.processUnit(context.Background(), tt.unit)
			if err != nil {
				t.Fatalf("processUnit() error = %v", err)
			}

			// Check that the output file was created
			outputFile := filepath.Join(tempDir, tt.unit.Values[0].Filename+".yaml")
			if _, err := os.Stat(outputFile); os.IsNotExist(err) {
				t.Fatalf("Output file %s was not created", outputFile)
			}

			// Read and verify the content
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			// Parse the YAML to verify APIVersion and Kind
			var result struct {
				APIVersion string `yaml:"apiVersion"`
				Kind       string `yaml:"kind"`
			}

			err = kyaml.Unmarshal(content, &result)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			if result.APIVersion != tt.expectedAPIVersion {
				t.Errorf("Expected APIVersion %s, got %s", tt.expectedAPIVersion, result.APIVersion)
			}

			if result.Kind != tt.expectedKind {
				t.Errorf("Expected Kind %s, got %s", tt.expectedKind, result.Kind)
			}

			// Clean up the test file
			err = os.Remove(outputFile)
			if err != nil {
				t.Fatalf("Failed to remove test output file: %v", err)
			}
		})
	}
}
