package types

import (
	"strings"
	"testing"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCleanCronWorkflow_ToYAMLWithIndent(t *testing.T) {
	// Create test CronWorkflow
	cw := &argoworkflowsv1alpha1.CronWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "CronWorkflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-workflow",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
			Schedule: "0 0 * * *",
			WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
				Entrypoint: "main",
			},
		},
	}

	cleanCW := NewCleanCronWorkflow(cw)

	tests := []struct {
		name               string
		indent             int
		expectedIndentChar string
	}{
		{
			name:               "indent 1 space",
			indent:             1,
			expectedIndentChar: " ",
		},
		{
			name:               "indent 2 spaces",
			indent:             2,
			expectedIndentChar: "  ",
		},
		{
			name:               "indent 4 spaces",
			indent:             4,
			expectedIndentChar: "    ",
		},
		{
			name:               "indent 8 spaces",
			indent:             8,
			expectedIndentChar: "        ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cleanCW.ToYAMLWithIndent(tt.indent)
			assert.NoError(t, err)
			assert.NotEmpty(t, result)

			yamlString := string(result)

			// Check basic YAML structure
			assert.Contains(t, yamlString, "apiVersion: argoproj.io/v1alpha1")
			assert.Contains(t, yamlString, "kind: CronWorkflow")
			assert.Contains(t, yamlString, "name: test-workflow")
			assert.Contains(t, yamlString, "namespace: test-namespace")
			assert.Contains(t, yamlString, "schedule: 0 0 * * *")

			// Check indentation by looking for nested elements
			lines := strings.Split(yamlString, "\n")
			var foundNestedLine bool
			for _, line := range lines {
				// Look for nested elements like "name:" under metadata
				if strings.Contains(line, "name: test-workflow") {
					// This line should be indented from the root level
					if strings.HasPrefix(line, tt.expectedIndentChar) {
						foundNestedLine = true
						break
					}
				}
			}
			assert.True(t, foundNestedLine, "Expected to find properly indented nested line with %d spaces", tt.indent)
		})
	}
}

func TestCleanCronWorkflow_ToYAML_DefaultIndent(t *testing.T) {
	// Create minimal test CronWorkflow
	cw := &argoworkflowsv1alpha1.CronWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "CronWorkflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "simple-workflow",
		},
		Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
			Schedule: "0 0 * * *",
		},
	}

	cleanCW := NewCleanCronWorkflow(cw)

	// Test default ToYAML method
	defaultResult, err := cleanCW.ToYAML()
	assert.NoError(t, err)

	// Test ToYAMLWithIndent with indent 2
	customResult, err := cleanCW.ToYAMLWithIndent(2)
	assert.NoError(t, err)

	// They should be identical since default is 2 spaces
	assert.Equal(t, string(defaultResult), string(customResult))

	// Verify it contains expected content
	yamlString := string(defaultResult)
	assert.Contains(t, yamlString, "apiVersion: argoproj.io/v1alpha1")
	assert.Contains(t, yamlString, "kind: CronWorkflow")
	assert.Contains(t, yamlString, "name: simple-workflow")
	assert.Contains(t, yamlString, "schedule: 0 0 * * *")
}

func TestCleanCronWorkflow_ToYAML_EmptyMetadata(t *testing.T) {
	// Create CronWorkflow with no metadata
	cw := &argoworkflowsv1alpha1.CronWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "CronWorkflow",
		},
		Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
			Schedule: "0 0 * * *",
		},
	}

	cleanCW := NewCleanCronWorkflow(cw)
	result, err := cleanCW.ToYAMLWithIndent(2)
	assert.NoError(t, err)

	yamlString := string(result)

	// Should have apiVersion and kind
	assert.Contains(t, yamlString, "apiVersion: argoproj.io/v1alpha1")
	assert.Contains(t, yamlString, "kind: CronWorkflow")
	assert.Contains(t, yamlString, "schedule: 0 0 * * *")

	// Should not have metadata section
	assert.NotContains(t, yamlString, "metadata:")
}

func TestCleanCronWorkflow_ToYAML_CompareIndentSizes(t *testing.T) {
	// Create test CronWorkflow with nested structure
	cw := &argoworkflowsv1alpha1.CronWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "CronWorkflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "indent-test",
			Labels: map[string]string{
				"env": "test",
			},
		},
		Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
			Schedule: "0 0 * * *",
		},
	}

	cleanCW := NewCleanCronWorkflow(cw)

	// Generate YAML with different indents
	result2, err := cleanCW.ToYAMLWithIndent(2)
	assert.NoError(t, err)

	result4, err := cleanCW.ToYAMLWithIndent(4)
	assert.NoError(t, err)

	yaml2 := string(result2)
	yaml4 := string(result4)

	// Both should contain same content
	assert.Contains(t, yaml2, "name: indent-test")
	assert.Contains(t, yaml4, "name: indent-test")
	assert.Contains(t, yaml2, "env: test")
	assert.Contains(t, yaml4, "env: test")

	// But the indentation should be different
	assert.NotEqual(t, yaml2, yaml4)

	// Count spaces in front of "name:" (should be under metadata)
	lines2 := strings.Split(yaml2, "\n")
	lines4 := strings.Split(yaml4, "\n")

	var spaces2, spaces4 int
	for _, line := range lines2 {
		if strings.Contains(line, "name: indent-test") {
			spaces2 = len(line) - len(strings.TrimLeft(line, " "))
			break
		}
	}
	for _, line := range lines4 {
		if strings.Contains(line, "name: indent-test") {
			spaces4 = len(line) - len(strings.TrimLeft(line, " "))
			break
		}
	}

	// The 4-space version should have more indentation
	assert.Greater(t, spaces4, spaces2)
	assert.Equal(t, 2, spaces2)
	assert.Equal(t, 4, spaces4)
}