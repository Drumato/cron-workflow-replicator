package config

import (
	"errors"
	"os"
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
	files      map[string][]byte
	readErrors map[string]error
}

func NewMockFileReader() *MockFileReader {
	return &MockFileReader{
		files:      make(map[string][]byte),
		readErrors: make(map[string]error),
	}
}

func (m *MockFileReader) AddFile(path string, content []byte) {
	m.files[path] = content
}

func (m *MockFileReader) AddReadError(path string, err error) {
	m.readErrors[path] = err
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
	if err, exists := m.readErrors[filename]; exists {
		return nil, err
	}
	if content, exists := m.files[filename]; exists {
		return content, nil
	}
	return nil, os.ErrNotExist // File not found
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
		name              string
		baseManifestPath  *string
		configDir         string
		expectedFilePath  string
		shouldCreateFile  bool
		expectError       bool
		expectedName      string
		expectedNamespace string
		expectedSchedule  string
	}{
		{
			name:              "relative path resolution",
			baseManifestPath:  func() *string { s := "./base-manifest.yaml"; return &s }(),
			configDir:         "/config/dir",
			expectedFilePath:  "/config/dir/base-manifest.yaml",
			shouldCreateFile:  true,
			expectError:       false,
			expectedName:      "base-workflow",
			expectedNamespace: "base-namespace",
			expectedSchedule:  "0 0 * * *",
		},
		{
			name:              "absolute path (no resolution needed)",
			baseManifestPath:  func() *string { s := "/absolute/path/base-manifest.yaml"; return &s }(),
			configDir:         "/config/dir",
			expectedFilePath:  "/absolute/path/base-manifest.yaml",
			shouldCreateFile:  true,
			expectError:       false,
			expectedName:      "base-workflow",
			expectedNamespace: "base-namespace",
			expectedSchedule:  "0 0 * * *",
		},
		{
			name:              "nested relative path",
			baseManifestPath:  func() *string { s := "manifests/base.yaml"; return &s }(),
			configDir:         "/config/dir",
			expectedFilePath:  "/config/dir/manifests/base.yaml",
			shouldCreateFile:  true,
			expectError:       false,
			expectedName:      "base-workflow",
			expectedNamespace: "base-namespace",
			expectedSchedule:  "0 0 * * *",
		},
		{
			name:              "no base manifest path",
			baseManifestPath:  nil,
			configDir:         "/config/dir",
			expectedFilePath:  "",
			shouldCreateFile:  false,
			expectError:       false,
			expectedName:      "",
			expectedNamespace: "",
			expectedSchedule:  "",
		},
		{
			name:              "relative path file not found",
			baseManifestPath:  func() *string { s := "./nonexistent.yaml"; return &s }(),
			configDir:         "/config/dir",
			expectedFilePath:  "/config/dir/nonexistent.yaml",
			shouldCreateFile:  false,
			expectError:       true,
			expectedName:      "",
			expectedNamespace: "",
			expectedSchedule:  "",
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

func TestUnit_LoadBaseCronWorkflow_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name             string
		setupMock        func(*MockFileReader)
		baseManifestPath string
		configDir        string
		expectedError    string
	}{
		{
			name: "file permission denied",
			setupMock: func(mock *MockFileReader) {
				mock.AddReadError("/config/base.yaml", os.ErrPermission)
			},
			baseManifestPath: "base.yaml",
			configDir:        "/config",
			expectedError:    "failed to read base manifest file /config/base.yaml: permission denied",
		},
		{
			name: "file not found",
			setupMock: func(mock *MockFileReader) {
				// Don't add any files to simulate not found
			},
			baseManifestPath: "nonexistent.yaml",
			configDir:        "/config",
			expectedError:    "failed to read base manifest file /config/nonexistent.yaml: file does not exist",
		},
		{
			name: "invalid YAML syntax",
			setupMock: func(mock *MockFileReader) {
				invalidYAML := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: invalid-workflow
  invalid-yaml-syntax: [
    - incomplete: structure`
				mock.AddFile("/config/invalid.yaml", []byte(invalidYAML))
			},
			baseManifestPath: "invalid.yaml",
			configDir:        "/config",
			expectedError:    "failed to unmarshal base manifest file /config/invalid.yaml",
		},
		{
			name: "completely invalid YAML structure",
			setupMock: func(mock *MockFileReader) {
				invalidYAML := `{this is not valid YAML at all!!!
				random text with no structure
				]]]`
				mock.AddFile("/config/invalid.yaml", []byte(invalidYAML))
			},
			baseManifestPath: "invalid.yaml",
			configDir:        "/config",
			expectedError:    "failed to unmarshal base manifest file /config/invalid.yaml",
		},
		{
			name: "malformed YAML with mixed types",
			setupMock: func(mock *MockFileReader) {
				malformedYAML := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: 12345
  labels:
    - this-should-be-a-map-not-array
spec:
  schedule: "0 0 * * *"`
				mock.AddFile("/config/malformed.yaml", []byte(malformedYAML))
			},
			baseManifestPath: "malformed.yaml",
			configDir:        "/config",
			expectedError:    "failed to unmarshal base manifest file /config/malformed.yaml",
		},
		{
			name: "IO error during read",
			setupMock: func(mock *MockFileReader) {
				mock.AddReadError("/config/io-error.yaml", errors.New("IO error"))
			},
			baseManifestPath: "io-error.yaml",
			configDir:        "/config",
			expectedError:    "failed to read base manifest file /config/io-error.yaml: IO error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock file reader
			fileReader := NewMockFileReader()
			tt.setupMock(fileReader)

			// Create unit
			unit := Unit{
				BaseManifestPath: &tt.baseManifestPath,
				APIVersion:       APIVersionV1Alpha1,
			}

			// Execute
			result, err := unit.LoadBaseCronWorkflow(fileReader, tt.configDir)

			// Verify error
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			assert.Nil(t, result)
		})
	}
}

func TestDefaultFileReader_ErrorScenarios(t *testing.T) {
	reader := &DefaultFileReader{}

	tests := []struct {
		name         string
		filename     string
		expectedErr  string
	}{
		{
			name:        "nonexistent file",
			filename:    "/nonexistent/path/file.yaml",
			expectedErr: "no such file or directory",
		},
		{
			name:        "empty filename",
			filename:    "",
			expectedErr: "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := reader.ReadFile(tt.filename)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.Nil(t, data)
		})
	}
}
