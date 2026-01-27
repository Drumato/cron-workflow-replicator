package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIVersion_GetSchemeGroupVersion(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion APIVersion
		expected   string
	}{
		{
			name:       "v1alpha1 returns correct group version",
			apiVersion: APIVersionV1Alpha1,
			expected:   "argoproj.io/v1alpha1",
		},
		{
			name:       "empty string defaults to v1alpha1",
			apiVersion: "",
			expected:   "argoproj.io/v1alpha1",
		},
		{
			name:       "unknown version defaults to v1alpha1",
			apiVersion: "unknown",
			expected:   "argoproj.io/v1alpha1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.apiVersion.GetSchemeGroupVersion()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPIVersion_GetKind(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion APIVersion
		expected   string
	}{
		{
			name:       "v1alpha1 returns CronWorkflow",
			apiVersion: APIVersionV1Alpha1,
			expected:   "CronWorkflow",
		},
		{
			name:       "empty string defaults to CronWorkflow",
			apiVersion: "",
			expected:   "CronWorkflow",
		},
		{
			name:       "unknown version defaults to CronWorkflow",
			apiVersion: "unknown",
			expected:   "CronWorkflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.apiVersion.GetKind()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// MockFileReader for testing path resolution
type MockFileReader struct {
	files map[string][]byte
}

func NewMockFileReader() *MockFileReader {
	return &MockFileReader{
		files: make(map[string][]byte),
	}
}

func (m *MockFileReader) AddFile(path string, content []byte) {
	m.files[path] = content
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
	if content, exists := m.files[filename]; exists {
		return content, nil
	}
	return nil, assert.AnError // File not found
}

func TestUnit_LoadBaseCronWorkflow_PathResolution(t *testing.T) {
	baseManifestContent := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: base-workflow
  namespace: base-namespace
spec:
  schedule: "0 0 * * *"
  workflowSpec:
    entrypoint: main`

	tests := []struct {
		name               string
		baseManifestPath   *string
		configDir          string
		expectedFilePath   string
		shouldCreateFile   bool
		expectError        bool
		expectedName       string
		expectedNamespace  string
		expectedSchedule   string
	}{
		{
			name:               "relative path resolution",
			baseManifestPath:   func() *string { s := "./base-manifest.yaml"; return &s }(),
			configDir:          "/config/dir",
			expectedFilePath:   "/config/dir/base-manifest.yaml",
			shouldCreateFile:   true,
			expectError:        false,
			expectedName:       "base-workflow",
			expectedNamespace:  "base-namespace",
			expectedSchedule:   "0 0 * * *",
		},
		{
			name:               "absolute path (no resolution needed)",
			baseManifestPath:   func() *string { s := "/absolute/path/base-manifest.yaml"; return &s }(),
			configDir:          "/config/dir",
			expectedFilePath:   "/absolute/path/base-manifest.yaml",
			shouldCreateFile:   true,
			expectError:        false,
			expectedName:       "base-workflow",
			expectedNamespace:  "base-namespace",
			expectedSchedule:   "0 0 * * *",
		},
		{
			name:               "nested relative path",
			baseManifestPath:   func() *string { s := "manifests/base.yaml"; return &s }(),
			configDir:          "/config/dir",
			expectedFilePath:   "/config/dir/manifests/base.yaml",
			shouldCreateFile:   true,
			expectError:        false,
			expectedName:       "base-workflow",
			expectedNamespace:  "base-namespace",
			expectedSchedule:   "0 0 * * *",
		},
		{
			name:               "no base manifest path",
			baseManifestPath:   nil,
			configDir:          "/config/dir",
			expectedFilePath:   "",
			shouldCreateFile:   false,
			expectError:        false,
			expectedName:       "",
			expectedNamespace:  "",
			expectedSchedule:   "",
		},
		{
			name:               "relative path file not found",
			baseManifestPath:   func() *string { s := "./nonexistent.yaml"; return &s }(),
			configDir:          "/config/dir",
			expectedFilePath:   "/config/dir/nonexistent.yaml",
			shouldCreateFile:   false,
			expectError:        true,
			expectedName:       "",
			expectedNamespace:  "",
			expectedSchedule:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock file reader
			fileReader := NewMockFileReader()
			if tt.shouldCreateFile {
				fileReader.AddFile(tt.expectedFilePath, []byte(baseManifestContent))
			}

			// Create unit
			unit := Unit{
				BaseManifestPath: tt.baseManifestPath,
				APIVersion:       APIVersionV1Alpha1,
			}

			// Execute
			result, err := unit.LoadBaseCronWorkflow(fileReader, tt.configDir)

			// Verify error expectation
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Verify result
			assert.NotNil(t, result)
			assert.Equal(t, "argoproj.io/v1alpha1", result.APIVersion)
			assert.Equal(t, "CronWorkflow", result.Kind)

			if tt.baseManifestPath == nil {
				// No base manifest case - should return empty workflow with proper TypeMeta
				assert.Equal(t, "", result.Name)
				assert.Equal(t, "", result.Namespace)
				assert.Equal(t, "", result.Spec.Schedule)
			} else {
				// Base manifest case - should have loaded values
				assert.Equal(t, tt.expectedName, result.Name)
				assert.Equal(t, tt.expectedNamespace, result.Namespace)
				assert.Equal(t, tt.expectedSchedule, result.Spec.Schedule)
			}
		})
	}
}
