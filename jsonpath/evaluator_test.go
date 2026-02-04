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
		expected  []string
		expectErr bool
	}{
		{
			name:     "simple root path",
			path:     "$.name",
			expected: []string{"name"},
		},
		{
			name:     "nested path",
			path:     "$.metadata.name",
			expected: []string{"metadata", "name"},
		},
		{
			name:     "deep nested path",
			path:     "$.metadata.labels.app",
			expected: []string{"metadata", "labels", "app"},
		},
		{
			name:     "root only",
			path:     "$",
			expected: []string{},
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
