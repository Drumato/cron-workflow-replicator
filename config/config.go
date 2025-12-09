package config

import (
	"fmt"
	"io"
	"os"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kyaml "sigs.k8s.io/yaml"
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

// FileReader interface for reading files (allows dependency injection for testing)
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

// DefaultFileReader implements FileReader using standard library
type DefaultFileReader struct{}

func (dfr *DefaultFileReader) ReadFile(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close() // In read-only context, close errors are typically not actionable
	}()
	return io.ReadAll(file)
}

// LoadBaseCronWorkflow loads a CronWorkflow from the base manifest file if BaseManifestPath is provided
func (u *Unit) LoadBaseCronWorkflow(fileReader FileReader) (*argoworkflowsv1alpha1.CronWorkflow, error) {
	if u.BaseManifestPath == nil {
		// No base manifest, return empty CronWorkflow with proper TypeMeta
		return &argoworkflowsv1alpha1.CronWorkflow{
			TypeMeta: metav1.TypeMeta{
				APIVersion: u.APIVersion.GetSchemeGroupVersion(),
				Kind:       u.APIVersion.GetKind(),
			},
		}, nil
	}

	// Read the base manifest file
	data, err := fileReader.ReadFile(*u.BaseManifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base manifest file %s: %w", *u.BaseManifestPath, err)
	}

	// Unmarshal the YAML into CronWorkflow
	var baseCronWorkflow argoworkflowsv1alpha1.CronWorkflow
	if err := kyaml.Unmarshal(data, &baseCronWorkflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base manifest file %s: %w", *u.BaseManifestPath, err)
	}

	return &baseCronWorkflow, nil
}
