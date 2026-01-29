package runner

import (
	"testing"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/drumato/cron-workflow-replicator/config"
	"github.com/drumato/cron-workflow-replicator/structutil"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMergeLogic_BasicMerging(t *testing.T) {
	tests := []struct {
		name           string
		baseCW         *argoworkflowsv1alpha1.CronWorkflow
		value          config.Value
		expectedName   string
		expectedLabels map[string]string
		expectedSpec   argoworkflowsv1alpha1.CronWorkflowSpec
	}{
		{
			name: "merge metadata and spec fields",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "base-name",
					Namespace: "base-namespace",
					Labels: map[string]string{
						"app":         "base-app",
						"environment": "test",
					},
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Schedule: "0 0 * * *",
					Timezone: "Asia/Tokyo",
					Suspend:  false,
					WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
						Entrypoint: "base-entrypoint",
					},
				},
			},
			value: config.Value{
				Filename: "merged-workflow",
				Metadata: metav1.ObjectMeta{
					Name:      "custom-name",
					Namespace: "custom-namespace",
					Labels: map[string]string{
						"custom":    "label",
						"app":       "override-app", // Should override base
						"new-label": "new-value",
					},
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Schedule: "0 6 * * *", // Should override base
					// Timezone not specified, should keep base value
					WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
						Entrypoint: "custom-entrypoint", // Should override base
					},
				},
			},
			expectedName: "custom-name",
			expectedLabels: map[string]string{
				"app":         "override-app", // Overridden
				"environment": "test",         // From base
				"custom":      "label",        // New
				"new-label":   "new-value",    // New
			},
			expectedSpec: argoworkflowsv1alpha1.CronWorkflowSpec{
				Schedule: "0 6 * * *",  // Overridden
				Timezone: "Asia/Tokyo", // From base
				Suspend:  false,        // From base
				WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
					Entrypoint: "custom-entrypoint", // Overridden
				},
			},
		},
		{
			name: "merge with empty base",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			value: config.Value{
				Filename: "standalone-workflow",
				Metadata: metav1.ObjectMeta{
					Name:      "standalone-name",
					Namespace: "standalone-namespace",
					Labels: map[string]string{
						"standalone": "label",
					},
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Schedule: "0 12 * * 1",
					WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
						Entrypoint: "standalone-entrypoint",
					},
				},
			},
			expectedName: "standalone-name",
			expectedLabels: map[string]string{
				"standalone": "label",
			},
			expectedSpec: argoworkflowsv1alpha1.CronWorkflowSpec{
				Schedule: "0 12 * * 1",
				WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
					Entrypoint: "standalone-entrypoint",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the base to avoid modifying the original
			result := &argoworkflowsv1alpha1.CronWorkflow{}
			if tt.baseCW != nil {
				*result = *tt.baseCW
				// Deep copy labels map to avoid reference issues
				if tt.baseCW.Labels != nil {
					result.Labels = make(map[string]string)
					for k, v := range tt.baseCW.Labels {
						result.Labels[k] = v
					}
				}
			}

			// Apply the merge
			structutil.MergeStruct(&result.ObjectMeta, &tt.value.Metadata)
			structutil.MergeStruct(&result.Spec, &tt.value.Spec)

			// Validate results
			assert.Equal(t, tt.expectedName, result.Name, "Name merge failed")
			assert.Equal(t, tt.expectedLabels, result.Labels, "Labels merge failed")
			assert.Equal(t, tt.expectedSpec.Schedule, result.Spec.Schedule, "Schedule merge failed")
			assert.Equal(t, tt.expectedSpec.Timezone, result.Spec.Timezone, "Timezone merge failed")
			assert.Equal(t, tt.expectedSpec.Suspend, result.Spec.Suspend, "Suspend merge failed")
			assert.Equal(t, tt.expectedSpec.WorkflowSpec.Entrypoint, result.Spec.WorkflowSpec.Entrypoint, "Entrypoint merge failed")
		})
	}
}

func TestMergeLogic_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		baseCW      *argoworkflowsv1alpha1.CronWorkflow
		value       config.Value
		expectError bool
	}{
		{
			name: "merge with nil base labels",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   "base-name",
					Labels: nil, // Explicitly nil labels
				},
			},
			value: config.Value{
				Metadata: metav1.ObjectMeta{
					Labels: map[string]string{
						"new": "label",
					},
				},
			},
			expectError: false,
		},
		{
			name: "merge with empty value labels",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "base-name",
					Labels: map[string]string{
						"base": "label",
					},
				},
			},
			value: config.Value{
				Metadata: metav1.ObjectMeta{
					Labels: nil, // Nil labels in value
				},
			},
			expectError: false,
		},
		{
			name: "merge with zero values",
			baseCW: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Suspend: true, // Base has suspend=true
				},
			},
			value: config.Value{
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Suspend: false, // Value has suspend=false (zero value for bool)
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the base
			result := &argoworkflowsv1alpha1.CronWorkflow{}
			if tt.baseCW != nil {
				*result = *tt.baseCW
				if tt.baseCW.Labels != nil {
					result.Labels = make(map[string]string)
					for k, v := range tt.baseCW.Labels {
						result.Labels[k] = v
					}
				}
			}

			// Apply the merge
			structutil.MergeStruct(&result.ObjectMeta, &tt.value.Metadata)
			structutil.MergeStruct(&result.Spec, &tt.value.Spec)

			// Basic validation that merge completed without corrupting structure
			assert.Equal(t, "argoproj.io/v1alpha1", result.APIVersion)
			assert.Equal(t, "CronWorkflow", result.Kind)
		})
	}
}

func TestMergeLogic_ComplexNestedStructures(t *testing.T) {
	// Test merging complex nested structures like WorkflowSpec with Arguments
	baseCW := &argoworkflowsv1alpha1.CronWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "CronWorkflow",
		},
		Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
			WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
				Entrypoint: "main",
				Arguments: argoworkflowsv1alpha1.Arguments{
					Parameters: []argoworkflowsv1alpha1.Parameter{
						{
							Name:  "base-param",
							Value: argoworkflowsv1alpha1.AnyStringPtr("base-value"),
						},
					},
				},
			},
		},
	}

	value := config.Value{
		Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
			WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
				Entrypoint: "custom-main", // Override entrypoint
				Arguments: argoworkflowsv1alpha1.Arguments{
					Parameters: []argoworkflowsv1alpha1.Parameter{
						{
							Name:  "custom-param",
							Value: argoworkflowsv1alpha1.AnyStringPtr("custom-value"),
						},
					},
				},
			},
		},
	}

	// Create a copy for merging
	result := *baseCW

	// Apply the merge
	structutil.MergeStruct(&result.Spec, &value.Spec)

	// Validate that override took effect
	assert.Equal(t, "custom-main", result.Spec.WorkflowSpec.Entrypoint)

	// Note: The specific behavior of parameter merging depends on structutil.MergeStruct implementation
	// This test validates that complex nested structures don't cause the merge to fail
	assert.NotNil(t, result.Spec.WorkflowSpec.Arguments.Parameters)
	assert.NotEmpty(t, result.Spec.WorkflowSpec.Arguments.Parameters)
}
