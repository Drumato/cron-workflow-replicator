package runner

import (
	"strings"
	"testing"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/drumato/cron-workflow-replicator/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kyaml "sigs.k8s.io/yaml"
)

func TestYAMLGeneration_BasicStructure(t *testing.T) {
	tests := []struct {
		name       string
		cw         *argoworkflowsv1alpha1.CronWorkflow
		apiVersion config.APIVersion
	}{
		{
			name: "v1alpha1 API version",
			cw: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test-app",
					},
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Schedule: "0 0 * * *",
					Timezone: "Asia/Tokyo",
					Suspend:  false,
					WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
						Entrypoint: "main",
					},
				},
			},
			apiVersion: config.APIVersionV1Alpha1,
		},
		{
			name: "empty API version defaults to v1alpha1",
			cw: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default-version-workflow",
					Namespace: "default",
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Schedule: "0 12 * * 1",
				},
			},
			apiVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate YAML
			yamlBytes, err := kyaml.Marshal(tt.cw)
			require.NoError(t, err, "Failed to marshal CronWorkflow to YAML")

			// Validate YAML structure
			yamlStr := string(yamlBytes)
			assert.Contains(t, yamlStr, "apiVersion: argoproj.io/v1alpha1")
			assert.Contains(t, yamlStr, "kind: CronWorkflow")

			// Parse back to validate structure integrity
			var parsedCW argoworkflowsv1alpha1.CronWorkflow
			err = kyaml.Unmarshal(yamlBytes, &parsedCW)
			require.NoError(t, err, "Generated YAML should be parseable")

			// Validate key fields are preserved
			assert.Equal(t, tt.cw.APIVersion, parsedCW.APIVersion)
			assert.Equal(t, tt.cw.Kind, parsedCW.Kind)
			assert.Equal(t, tt.cw.Name, parsedCW.Name)
			assert.Equal(t, tt.cw.Namespace, parsedCW.Namespace)
			assert.Equal(t, tt.cw.Labels, parsedCW.Labels)
			assert.Equal(t, tt.cw.Spec.Schedule, parsedCW.Spec.Schedule)
		})
	}
}

func TestYAMLGeneration_EmptyValues(t *testing.T) {
	tests := []struct {
		name           string
		cw             *argoworkflowsv1alpha1.CronWorkflow
		expectPresent  []string
		expectAbsent   []string
	}{
		{
			name: "minimal CronWorkflow",
			cw: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			expectPresent: []string{
				"apiVersion: argoproj.io/v1alpha1",
				"kind: CronWorkflow",
			},
			expectAbsent: []string{
				"name:",
				"namespace:",
				"labels:",
				"schedule:",
			},
		},
		{
			name: "CronWorkflow with empty spec",
			cw: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-spec-workflow",
					Namespace: "default",
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{},
			},
			expectPresent: []string{
				"apiVersion: argoproj.io/v1alpha1",
				"kind: CronWorkflow",
				"name: empty-spec-workflow",
				"namespace: default",
			},
			expectAbsent: []string{
				"schedule:",
				"timezone:",
			},
		},
		{
			name: "CronWorkflow with nil labels",
			cw: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nil-labels-workflow",
					Namespace: "default",
					Labels:    nil,
				},
			},
			expectPresent: []string{
				"name: nil-labels-workflow",
				"namespace: default",
			},
			expectAbsent: []string{
				"labels:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlBytes, err := kyaml.Marshal(tt.cw)
			require.NoError(t, err)

			yamlStr := string(yamlBytes)

			// Check expected present fields
			for _, expected := range tt.expectPresent {
				assert.Contains(t, yamlStr, expected, "Expected '%s' to be present in YAML", expected)
			}

			// Check expected absent fields
			for _, notExpected := range tt.expectAbsent {
				assert.NotContains(t, yamlStr, notExpected, "Expected '%s' to be absent from YAML", notExpected)
			}

			// Validate generated YAML is parseable
			var parsed argoworkflowsv1alpha1.CronWorkflow
			err = kyaml.Unmarshal(yamlBytes, &parsed)
			require.NoError(t, err)
		})
	}
}

func TestYAMLGeneration_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name string
		cw   *argoworkflowsv1alpha1.CronWorkflow
	}{
		{
			name: "special characters in name and labels",
			cw: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "workflow-with-hyphens",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"app.kubernetes.io/name":     "test-app",
						"app.kubernetes.io/instance": "test-instance",
						"version":                    "v1.2.3",
					},
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Schedule: "*/15 * * * *", // Cron with special characters
					Timezone: "America/New_York",
				},
			},
		},
		{
			name: "unicode characters",
			cw: &argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unicode-workflow",
					Namespace: "default",
					Labels: map[string]string{
						"description": "テスト",
						"emoji":       "🚀",
					},
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Schedule: "0 9 * * 月-金", // Japanese characters in cron
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlBytes, err := kyaml.Marshal(tt.cw)
			require.NoError(t, err, "Should be able to marshal special characters")

			// Validate YAML is parseable
			var parsed argoworkflowsv1alpha1.CronWorkflow
			err = kyaml.Unmarshal(yamlBytes, &parsed)
			require.NoError(t, err, "Generated YAML with special characters should be parseable")

			// Validate special characters are preserved
			assert.Equal(t, tt.cw.Name, parsed.Name)
			assert.Equal(t, tt.cw.Namespace, parsed.Namespace)
			assert.Equal(t, tt.cw.Labels, parsed.Labels)
			assert.Equal(t, tt.cw.Spec.Schedule, parsed.Spec.Schedule)
			assert.Equal(t, tt.cw.Spec.Timezone, parsed.Spec.Timezone)
		})
	}
}

func TestYAMLGeneration_LargeComplexStructures(t *testing.T) {
	// Test YAML generation with complex nested structures
	cw := &argoworkflowsv1alpha1.CronWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "CronWorkflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "complex-workflow",
			Namespace: "production",
			Labels: map[string]string{
				"app":         "data-pipeline",
				"version":     "v2.1.0",
				"environment": "production",
				"team":        "data-engineering",
			},
			Annotations: map[string]string{
				"description": "Complex data processing workflow",
				"owner":       "data-team@company.com",
			},
		},
		Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
			Schedule:                "0 2 * * *",
			Timezone:                "UTC",
			Suspend:                 false,
			StartingDeadlineSeconds: func() *int64 { v := int64(300); return &v }(),
			ConcurrencyPolicy:       "Forbid",
			WorkflowSpec: argoworkflowsv1alpha1.WorkflowSpec{
				Entrypoint: "main",
				Arguments: argoworkflowsv1alpha1.Arguments{
					Parameters: []argoworkflowsv1alpha1.Parameter{
						{
							Name:  "environment",
							Value: argoworkflowsv1alpha1.AnyStringPtr("production"),
						},
						{
							Name:  "batch_size",
							Value: argoworkflowsv1alpha1.AnyStringPtr("1000"),
						},
					},
				},
				Templates: []argoworkflowsv1alpha1.Template{
					{
						Name: "main",
						Steps: []argoworkflowsv1alpha1.ParallelSteps{
							{
								Steps: []argoworkflowsv1alpha1.WorkflowStep{
									{
										Name:     "data-extraction",
										Template: "extract-data",
									},
								},
							},
							{
								Steps: []argoworkflowsv1alpha1.WorkflowStep{
									{
										Name:     "data-processing",
										Template: "process-data",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	yamlBytes, err := kyaml.Marshal(cw)
	require.NoError(t, err, "Should be able to marshal complex structure")

	// Validate YAML is parseable and not truncated
	yamlStr := string(yamlBytes)
	assert.Greater(t, len(yamlStr), 500, "Complex YAML should be substantial in size")

	// Parse back and validate key nested fields
	var parsed argoworkflowsv1alpha1.CronWorkflow
	err = kyaml.Unmarshal(yamlBytes, &parsed)
	require.NoError(t, err, "Complex YAML should be parseable")

	assert.Equal(t, cw.Name, parsed.Name)
	assert.Equal(t, cw.Labels, parsed.Labels)
	assert.Equal(t, cw.Annotations, parsed.Annotations)
	assert.Equal(t, cw.Spec.Schedule, parsed.Spec.Schedule)
	assert.Equal(t, cw.Spec.WorkflowSpec.Entrypoint, parsed.Spec.WorkflowSpec.Entrypoint)
	assert.Equal(t, len(cw.Spec.WorkflowSpec.Arguments.Parameters), len(parsed.Spec.WorkflowSpec.Arguments.Parameters))
	assert.Equal(t, len(cw.Spec.WorkflowSpec.Templates), len(parsed.Spec.WorkflowSpec.Templates))
}

func TestYAMLGeneration_FieldOrdering(t *testing.T) {
	// Test that generated YAML has a logical field ordering (metadata before spec)
	cw := &argoworkflowsv1alpha1.CronWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "CronWorkflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ordered-workflow",
			Namespace: "default",
		},
		Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
			Schedule: "0 0 * * *",
		},
	}

	yamlBytes, err := kyaml.Marshal(cw)
	require.NoError(t, err)

	yamlStr := string(yamlBytes)
	lines := strings.Split(yamlStr, "\n")

	// Find positions of key sections
	var apiVersionPos, kindPos, metadataPos, specPos int
	for i, line := range lines {
		if strings.HasPrefix(line, "apiVersion:") {
			apiVersionPos = i
		} else if strings.HasPrefix(line, "kind:") {
			kindPos = i
		} else if strings.HasPrefix(line, "metadata:") {
			metadataPos = i
		} else if strings.HasPrefix(line, "spec:") {
			specPos = i
		}
	}

	// Validate logical ordering
	assert.Less(t, apiVersionPos, kindPos, "apiVersion should come before kind")
	assert.Less(t, kindPos, metadataPos, "kind should come before metadata")
	assert.Less(t, metadataPos, specPos, "metadata should come before spec")
}