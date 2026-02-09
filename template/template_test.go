package template

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateRenderer_RenderConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	renderer := New(logger)

	// Create temporary files for testing
	tempDir := t.TempDir()

	configContent := `units:
  - outputDirectory: "{{.Var.outputBase}}-{{.Var.environment}}"
    apiVersion: "{{.Var.apiVersion}}"
    values:
      - filename: "{{.Var.jobName}}-cronworkflow"
        paths:
          - path: "$.metadata.name"
            value: "{{.Var.jobName}}-cronworkflow"`

	valuesContent := `apiVersion: "v1alpha1"
outputBase: "./output"
environment: "test"
jobName: "sample-job"`

	configPath := filepath.Join(tempDir, "config.yaml")
	valuesPath := filepath.Join(tempDir, "values.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to create values file: %v", err)
	}

	// Test rendering
	result, err := renderer.RenderConfig(configPath, valuesPath)
	if err != nil {
		t.Fatalf("RenderConfig failed: %v", err)
	}

	resultStr := string(result)

	// Check that template variables were replaced
	if strings.Contains(resultStr, "{{") || strings.Contains(resultStr, "}}") {
		t.Errorf("Template variables were not replaced: %s", resultStr)
	}

	expectedStrings := []string{
		"outputDirectory: \"./output-test\"",
		"apiVersion: \"v1alpha1\"",
		"filename: \"sample-job-cronworkflow\"",
		"value: \"sample-job-cronworkflow\"",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(resultStr, expected) {
			t.Errorf("Expected string not found in result: %s\nActual result: %s", expected, resultStr)
		}
	}
}

func TestTemplateRenderer_RenderConfigFromReader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	renderer := New(logger)

	// Create temporary values file
	tempDir := t.TempDir()
	valuesContent := `name: "test-app"
version: "1.0.0"`

	valuesPath := filepath.Join(tempDir, "values.yaml")
	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to create values file: %v", err)
	}

	// Test with config from reader
	configContent := `app: {{.Var.name}}
version: {{.Var.version}}`

	configReader := strings.NewReader(configContent)

	result, err := renderer.RenderConfigFromReader(configReader, valuesPath)
	if err != nil {
		t.Fatalf("RenderConfigFromReader failed: %v", err)
	}

	resultStr := string(result)

	expectedStrings := []string{
		"app: test-app",
		"version: 1.0.0",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(resultStr, expected) {
			t.Errorf("Expected string not found in result: %s\nActual result: %s", expected, resultStr)
		}
	}
}

func TestTemplateRenderer_RenderConfig_FileNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	renderer := New(logger)

	// Test with non-existent config file
	_, err := renderer.RenderConfig("/nonexistent/config.yaml", "/nonexistent/values.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file, but got nil")
	}
}

func TestTemplateRenderer_RenderConfig_InvalidTemplate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	renderer := New(logger)

	tempDir := t.TempDir()

	// Create config with invalid template syntax
	configContent := `units:
  - name: {{.Var.name
    invalid: template`

	valuesContent := `name: "test"`

	configPath := filepath.Join(tempDir, "config.yaml")
	valuesPath := filepath.Join(tempDir, "values.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to create values file: %v", err)
	}

	_, err := renderer.RenderConfig(configPath, valuesPath)
	if err == nil {
		t.Error("Expected error for invalid template, but got nil")
	}
}

func TestTemplateRenderer_RenderConfig_InvalidValuesFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	renderer := New(logger)

	tempDir := t.TempDir()

	configContent := `name: {{.Var.name}}`
	valuesContent := `invalid: yaml: content: [[[`

	configPath := filepath.Join(tempDir, "config.yaml")
	valuesPath := filepath.Join(tempDir, "values.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to create values file: %v", err)
	}

	_, err := renderer.RenderConfig(configPath, valuesPath)
	if err == nil {
		t.Error("Expected error for invalid values file, but got nil")
	}
}

func TestHasTemplateVars(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "has template vars",
			content:  "name: {{.Var.name}}",
			expected: true,
		},
		{
			name:     "no template vars",
			content:  "name: test-app",
			expected: false,
		},
		{
			name:     "has opening but no closing",
			content:  "name: {{.Var.name",
			expected: false,
		},
		{
			name:     "has closing but no opening",
			content:  "name: .Var.name}}",
			expected: false,
		},
		{
			name:     "empty content",
			content:  "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := HasTemplateVars(tc.content)
			if result != tc.expected {
				t.Errorf("HasTemplateVars(%q) = %v, expected %v", tc.content, result, tc.expected)
			}
		})
	}
}