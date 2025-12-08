package config

import (
	"testing"
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
			if result != tt.expected {
				t.Errorf("GetSchemeGroupVersion() = %v, expected %v", result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("GetKind() = %v, expected %v", result, tt.expected)
			}
		})
	}
}