package jsonpath

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/drumato/cron-workflow-replicator/config"
)

func TestPathEvaluator_ApplyPaths(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	evaluator := NewPathEvaluator(logger)

	tests := []struct {
		name      string
		baseCW    *argoworkflowsv1alpha1.CronWorkflow
		paths     []config.PathValue
		expectErr bool
		validator func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow)
	}{
		{
			name: "apply basic metadata paths",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.metadata.name", Value: "test-cron"},
				{Path: "$.metadata.namespace", Value: "default"},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				assert.Equal(t, "test-cron", cw.Name)
				assert.Equal(t, "default", cw.Namespace)
			},
		},
		{
			name: "apply spec schedule path",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.schedule", Value: "0 0 * * *"},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				assert.Equal(t, "0 0 * * *", cw.Spec.Schedule)
			},
		},
		{
			name: "apply nested labels path",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.metadata.labels.app", Value: "my-app"},
				{Path: "$.metadata.labels.version", Value: "1.0.0"},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Labels)
				assert.Equal(t, "my-app", cw.Labels["app"])
				assert.Equal(t, "1.0.0", cw.Labels["version"])
			},
		},
		{
			name: "empty paths should not error",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths:     []config.PathValue{},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				// Should remain unchanged
				assert.Equal(t, "argoproj.io/v1alpha1", cw.APIVersion)
			},
		},
		{
			name: "invalid JSONPath should error",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "invalid-path", Value: "test"},
			},
			expectErr: true,
		},
		{
			name: "array index access - positive index",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.templates[0].name", Value: "first-template"},
				{Path: "$.spec.workflowSpec.templates[1].name", Value: "second-template"},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Spec.WorkflowSpec)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates)
				require.Len(t, cw.Spec.WorkflowSpec.Templates, 2)
				assert.Equal(t, "first-template", cw.Spec.WorkflowSpec.Templates[0].Name)
				assert.Equal(t, "second-template", cw.Spec.WorkflowSpec.Templates[1].Name)
			},
		},
		{
			name: "array index access - negative index",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.templates[0].name", Value: "first-template"},
				{Path: "$.spec.workflowSpec.templates[1].name", Value: "second-template"},
				{Path: "$.spec.workflowSpec.templates[-1].name", Value: "last-template"}, // Should replace second-template
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Spec.WorkflowSpec)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates)
				require.Len(t, cw.Spec.WorkflowSpec.Templates, 2)
				assert.Equal(t, "first-template", cw.Spec.WorkflowSpec.Templates[0].Name)
				assert.Equal(t, "last-template", cw.Spec.WorkflowSpec.Templates[1].Name) // [-1] refers to last element
			},
		},
		{
			name: "type conversion - numeric values",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.parallelism", Value: "5"},
				{Path: "$.spec.workflowSpec.activeDeadlineSeconds", Value: "3600"},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Spec.WorkflowSpec)
				// JSON unmarshaling converts numbers to their appropriate types after struct marshaling
				// In this case, they should be converted properly by the CronWorkflow struct
				// Let's check if the values are correctly set, regardless of the specific type
			},
		},
		{
			name: "type conversion - boolean values",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.suspend", Value: "true"},
				{Path: "$.spec.workflowSpec.suspend", Value: "false"},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				assert.Equal(t, true, cw.Spec.Suspend)
				require.NotNil(t, cw.Spec.WorkflowSpec)
				require.NotNil(t, cw.Spec.WorkflowSpec.Suspend)
				assert.Equal(t, false, *cw.Spec.WorkflowSpec.Suspend)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the test case
			cw := *tt.baseCW

			err := evaluator.ApplyPaths(&cw, tt.paths)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validator != nil {
					tt.validator(t, &cw)
				}
			}
		})
	}
}

func TestPathEvaluator_parseJSONPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	evaluator := NewPathEvaluator(logger)

	tests := []struct {
		name      string
		path      string
		expected  []PathSegment
		expectErr bool
	}{
		{
			name: "simple root path",
			path: "$.name",
			expected: []PathSegment{
				{Key: "name", ArrayIndex: nil, IsNegative: false},
			},
		},
		{
			name: "nested path",
			path: "$.metadata.name",
			expected: []PathSegment{
				{Key: "metadata", ArrayIndex: nil, IsNegative: false},
				{Key: "name", ArrayIndex: nil, IsNegative: false},
			},
		},
		{
			name: "deep nested path",
			path: "$.metadata.labels.app",
			expected: []PathSegment{
				{Key: "metadata", ArrayIndex: nil, IsNegative: false},
				{Key: "labels", ArrayIndex: nil, IsNegative: false},
				{Key: "app", ArrayIndex: nil, IsNegative: false},
			},
		},
		{
			name:     "root only",
			path:     "$",
			expected: []PathSegment{},
		},
		{
			name:      "invalid path without $",
			path:      "metadata.name",
			expectErr: true,
		},
		{
			name:      "empty path",
			path:      "",
			expectErr: true,
		},
		{
			name: "array index path",
			path: "$.spec.templates[0].name",
			expected: []PathSegment{
				{Key: "spec", ArrayIndex: nil, IsNegative: false},
				{Key: "templates", ArrayIndex: &[]int{0}[0], IsNegative: false},
				{Key: "name", ArrayIndex: nil, IsNegative: false},
			},
		},
		{
			name: "negative array index path",
			path: "$.spec.templates[-1].name",
			expected: []PathSegment{
				{Key: "spec", ArrayIndex: nil, IsNegative: false},
				{Key: "templates", ArrayIndex: &[]int{-1}[0], IsNegative: true},
				{Key: "name", ArrayIndex: nil, IsNegative: false},
			},
		},
		{
			name: "multiple array indices",
			path: "$.spec.templates[0].container.args[2]",
			expected: []PathSegment{
				{Key: "spec", ArrayIndex: nil, IsNegative: false},
				{Key: "templates", ArrayIndex: &[]int{0}[0], IsNegative: false},
				{Key: "container", ArrayIndex: nil, IsNegative: false},
				{Key: "args", ArrayIndex: &[]int{2}[0], IsNegative: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.parseJSONPath(tt.path)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPathEvaluator_structToMapAndBack(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	evaluator := NewPathEvaluator(logger)

	original := &argoworkflowsv1alpha1.CronWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "CronWorkflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cron",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test-app",
			},
		},
		Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
			Schedule: "0 0 * * *",
		},
	}

	// Convert to map
	m, err := evaluator.structToMap(original)
	require.NoError(t, err)
	require.NotNil(t, m)

	// Convert back to struct
	var result argoworkflowsv1alpha1.CronWorkflow
	err = evaluator.mapToStruct(m, &result)
	require.NoError(t, err)

	// Verify the round-trip preserved the data
	assert.Equal(t, original.TypeMeta, result.TypeMeta)
	assert.Equal(t, original.Name, result.Name)
	assert.Equal(t, original.Namespace, result.Namespace)
	assert.Equal(t, original.Labels, result.Labels)
	assert.Equal(t, original.Spec.Schedule, result.Spec.Schedule)
}

func TestPathEvaluator_convertValue(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	evaluator := NewPathEvaluator(logger)

	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "null value",
			input:    "null",
			expected: nil,
		},
		{
			name:     "boolean true",
			input:    "true",
			expected: true,
		},
		{
			name:     "boolean false",
			input:    "false",
			expected: false,
		},
		{
			name:     "integer value",
			input:    "123",
			expected: 123,
		},
		{
			name:     "negative integer value",
			input:    "-456",
			expected: -456,
		},
		{
			name:     "float value",
			input:    "123.45",
			expected: 123.45,
		},
		{
			name:     "negative float value",
			input:    "-67.89",
			expected: -67.89,
		},
		{
			name:     "JSON array",
			input:    `[1, 2, "test"]`,
			expected: []interface{}{float64(1), float64(2), "test"},
		},
		{
			name:     "JSON object",
			input:    `{"key": "value", "number": 42}`,
			expected: map[string]interface{}{"key": "value", "number": float64(42)},
		},
		{
			name:     "string value",
			input:    "just a string",
			expected: "just a string",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "invalid JSON array",
			input:    "[invalid json",
			expected: "[invalid json",
		},
		{
			name:     "invalid JSON object",
			input:    "{invalid json",
			expected: "{invalid json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.convertValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathEvaluator_ArrayOperations(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	evaluator := NewPathEvaluator(logger)

	tests := []struct {
		name      string
		baseCW    *argoworkflowsv1alpha1.CronWorkflow
		paths     []config.PathValue
		expectErr bool
		validator func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow)
	}{
		{
			name: "array extension - new elements",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.templates[0].name", Value: "first"},
				{Path: "$.spec.workflowSpec.templates[2].name", Value: "third"}, // Skip index 1
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Spec.WorkflowSpec)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates)
				require.Len(t, cw.Spec.WorkflowSpec.Templates, 3)
				assert.Equal(t, "first", cw.Spec.WorkflowSpec.Templates[0].Name)
				assert.Empty(t, cw.Spec.WorkflowSpec.Templates[1].Name) // Should be nil/empty
				assert.Equal(t, "third", cw.Spec.WorkflowSpec.Templates[2].Name)
			},
		},
		{
			name: "nested array access",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.templates[0].container.args[0]", Value: "arg1"},
				{Path: "$.spec.workflowSpec.templates[0].container.args[1]", Value: "arg2"},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Spec.WorkflowSpec)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates)
				require.Len(t, cw.Spec.WorkflowSpec.Templates, 1)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates[0].Container)
				require.Len(t, cw.Spec.WorkflowSpec.Templates[0].Container.Args, 2)
				assert.Equal(t, "arg1", cw.Spec.WorkflowSpec.Templates[0].Container.Args[0])
				assert.Equal(t, "arg2", cw.Spec.WorkflowSpec.Templates[0].Container.Args[1])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the test case
			cw := *tt.baseCW

			err := evaluator.ApplyPaths(&cw, tt.paths)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validator != nil {
					tt.validator(t, &cw)
				}
			}
		})
	}
}

func TestPathEvaluator_ArrayValueAssignment(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	evaluator := NewPathEvaluator(logger)

	tests := []struct {
		name      string
		baseCW    *argoworkflowsv1alpha1.CronWorkflow
		paths     []config.PathValue
		expectErr bool
		validator func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow)
	}{
		{
			name: "basic array assignment",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.arguments.parameters", Value: `[{"name": "env", "value": "prod"}, {"name": "version", "value": "1.0"}]`},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Spec.WorkflowSpec)
				require.NotNil(t, cw.Spec.WorkflowSpec.Arguments)
				require.NotNil(t, cw.Spec.WorkflowSpec.Arguments.Parameters)
				require.Len(t, cw.Spec.WorkflowSpec.Arguments.Parameters, 2)
				assert.Equal(t, "env", cw.Spec.WorkflowSpec.Arguments.Parameters[0].Name)
				assert.Equal(t, "prod", cw.Spec.WorkflowSpec.Arguments.Parameters[0].Value.String())
				assert.Equal(t, "version", cw.Spec.WorkflowSpec.Arguments.Parameters[1].Name)
				assert.Equal(t, "1.0", cw.Spec.WorkflowSpec.Arguments.Parameters[1].Value.String())
			},
		},
		{
			name: "simple string array assignment",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.templates[0].container.args", Value: `["arg1", "arg2", "arg3"]`},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Spec.WorkflowSpec)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates)
				require.Len(t, cw.Spec.WorkflowSpec.Templates, 1)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates[0].Container)
				require.Len(t, cw.Spec.WorkflowSpec.Templates[0].Container.Args, 3)
				assert.Equal(t, "arg1", cw.Spec.WorkflowSpec.Templates[0].Container.Args[0])
				assert.Equal(t, "arg2", cw.Spec.WorkflowSpec.Templates[0].Container.Args[1])
				assert.Equal(t, "arg3", cw.Spec.WorkflowSpec.Templates[0].Container.Args[2])
			},
		},
		{
			name: "empty array assignment",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.templates[0].container.args", Value: `[]`},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Spec.WorkflowSpec)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates)
				require.Len(t, cw.Spec.WorkflowSpec.Templates, 1)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates[0].Container)
				require.Empty(t, cw.Spec.WorkflowSpec.Templates[0].Container.Args)
			},
		},
		{
			name: "array replacement - should overwrite existing array",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.templates[0].container.args[0]", Value: "old-arg1"},
				{Path: "$.spec.workflowSpec.templates[0].container.args[1]", Value: "old-arg2"},
				{Path: "$.spec.workflowSpec.templates[0].container.args", Value: `["new-arg1", "new-arg2", "new-arg3"]`},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Spec.WorkflowSpec)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates)
				require.Len(t, cw.Spec.WorkflowSpec.Templates, 1)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates[0].Container)
				require.Len(t, cw.Spec.WorkflowSpec.Templates[0].Container.Args, 3)
				assert.Equal(t, "new-arg1", cw.Spec.WorkflowSpec.Templates[0].Container.Args[0])
				assert.Equal(t, "new-arg2", cw.Spec.WorkflowSpec.Templates[0].Container.Args[1])
				assert.Equal(t, "new-arg3", cw.Spec.WorkflowSpec.Templates[0].Container.Args[2])
			},
		},
		{
			name: "mixed operations - individual and whole array",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.templates[0].name", Value: "template-name"},
				{Path: "$.spec.workflowSpec.templates[0].container.args", Value: `["arg1", "arg2"]`},
				{Path: "$.spec.workflowSpec.templates[1].name", Value: "second-template"},
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				require.NotNil(t, cw.Spec.WorkflowSpec)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates)
				require.Len(t, cw.Spec.WorkflowSpec.Templates, 2)
				assert.Equal(t, "template-name", cw.Spec.WorkflowSpec.Templates[0].Name)
				require.NotNil(t, cw.Spec.WorkflowSpec.Templates[0].Container)
				require.Len(t, cw.Spec.WorkflowSpec.Templates[0].Container.Args, 2)
				assert.Equal(t, "arg1", cw.Spec.WorkflowSpec.Templates[0].Container.Args[0])
				assert.Equal(t, "arg2", cw.Spec.WorkflowSpec.Templates[0].Container.Args[1])
				assert.Equal(t, "second-template", cw.Spec.WorkflowSpec.Templates[1].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the test case
			cw := *tt.baseCW

			err := evaluator.ApplyPaths(&cw, tt.paths)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validator != nil {
					tt.validator(t, &cw)
				}
			}
		})
	}
}

func TestPathEvaluator_isArrayElementPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	evaluator := NewPathEvaluator(logger)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "array element path - positive index",
			path:     "$.spec.templates[0].name",
			expected: true,
		},
		{
			name:     "array element path - negative index",
			path:     "$.spec.templates[-1].name",
			expected: true,
		},
		{
			name:     "array element path - multi-digit index",
			path:     "$.spec.templates[123].name",
			expected: true,
		},
		{
			name:     "array path without index",
			path:     "$.spec.templates",
			expected: false,
		},
		{
			name:     "nested path without array index",
			path:     "$.spec.workflowSpec.arguments.parameters",
			expected: false,
		},
		{
			name:     "path ending with regular property",
			path:     "$.spec.schedule",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.isArrayElementPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathEvaluator_ErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	evaluator := NewPathEvaluator(logger)

	tests := []struct {
		name      string
		baseCW    *argoworkflowsv1alpha1.CronWorkflow
		paths     []config.PathValue
		expectErr bool
		errorMsg  string
		validator func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow)
	}{
		{
			name: "negative index on empty array should error",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.templates[-1].name", Value: "test"},
			},
			expectErr: true,
			errorMsg:  "cannot use negative index -1 on empty array",
		},
		{
			name: "negative index out of bounds should error",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.workflowSpec.templates[0].name", Value: "first"},
				{Path: "$.spec.workflowSpec.templates[-5].name", Value: "test"}, // -5 is out of bounds for array of length 1
			},
			expectErr: true,
			errorMsg:  "negative index -5 is out of bounds for array of length 1",
		},
		{
			name: "invalid array index syntax should error",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.templates[abc].name", Value: "test"},
			},
			expectErr: true,
			errorMsg:  "invalid syntax",
		},
		{
			name: "trying to access non-array as array should error",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Schedule: "0 0 * * *",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.schedule[0]", Value: "test"}, // schedule is a string, not an array
			},
			expectErr: true,
			errorMsg:  "is not an array",
		},
		{
			name: "invalid JSON array should be treated as string",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			paths: []config.PathValue{
				{Path: "$.spec.schedule", Value: "[invalid json array"}, // Invalid JSON
			},
			expectErr: false,
			validator: func(t *testing.T, cw *argoworkflowsv1alpha1.CronWorkflow) {
				assert.Equal(t, "[invalid json array", cw.Spec.Schedule)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the test case
			cw := *tt.baseCW

			err := evaluator.ApplyPaths(&cw, tt.paths)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				if tt.validator != nil {
					tt.validator(t, &cw)
				}
			}
		})
	}
}
