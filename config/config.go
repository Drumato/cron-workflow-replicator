package config

import (
	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type APIVersion string

const (
	APIVersionV1Alpha1 APIVersion = "v1alpha1"
	// APIVersionV1Beta1  APIVersion = "v1beta1"  // 未実装
	// APIVersionV1       APIVersion = "v1"       // 未実装
)

// GetSchemeGroupVersion returns the appropriate schema group version for the API version
func (av APIVersion) GetSchemeGroupVersion() string {
	switch av {
	case APIVersionV1Alpha1:
		return "argoproj.io/v1alpha1"
	default:
		// デフォルトはv1alpha1
		return "argoproj.io/v1alpha1"
	}
}

// GetKind returns the appropriate Kind for the API version
func (av APIVersion) GetKind() string {
	switch av {
	case APIVersionV1Alpha1:
		return "CronWorkflow"
	default:
		// デフォルトはCronWorkflow
		return "CronWorkflow"
	}
}

type Config struct {
	Units []Unit `yaml:"units"`
}

type Unit struct {
	BaseManifestPath *string    `yaml:"baseManifestPath"`
	OutputDirectory  string     `yaml:"outputDirectory"`
	APIVersion       APIVersion `yaml:"apiVersion"`
	Values           []Value    `yaml:"values"`
}

type Value struct {
	Filename string                                 `yaml:"filename"`
	Metadata metav1.ObjectMeta                      `yaml:"metadata"`
	Spec     argoworkflowsv1alpha1.CronWorkflowSpec `yaml:"spec"`
}
