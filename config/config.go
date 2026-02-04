package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

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
	BaseManifestPath *string          `yaml:"baseManifestPath"`
	OutputDirectory  string           `yaml:"outputDirectory"`
	APIVersion       APIVersion       `yaml:"apiVersion"`
	Kustomize        *KustomizeConfig `yaml:"kustomize"`
	Values           []Value          `yaml:"values"`
	Indent           *int             `yaml:"indent,omitempty"`
}

type KustomizeConfig struct {
	UpdateResources bool `yaml:"updateResources"`
}

type PathValue struct {
	Path  string `yaml:"path"`  // JSONPath式
	Value string `yaml:"value"` // 設定する文字列値
}

type Value struct {
	Filename string      `yaml:"filename"`
	Paths    []PathValue `yaml:"paths,omitempty"`
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
		if err := file.Close(); err != nil {
			slog.Warn("failed to close file", "filename", filename, "error", err)
		}
	}()
	return io.ReadAll(file)
}

// LoadBaseCronWorkflow loads a CronWorkflow from the base manifest file if BaseManifestPath is provided
func (u *Unit) LoadBaseCronWorkflow(fileReader FileReader, configDir string) (*argoworkflowsv1alpha1.CronWorkflow, error) {
	if u.BaseManifestPath == nil {
		// No base manifest, return empty CronWorkflow with proper TypeMeta
		return &argoworkflowsv1alpha1.CronWorkflow{
			TypeMeta: metav1.TypeMeta{
				APIVersion: u.APIVersion.GetSchemeGroupVersion(),
				Kind:       u.APIVersion.GetKind(),
			},
		}, nil
	}

	// Resolve relative path from config directory
	baseManifestPath := *u.BaseManifestPath
	if !filepath.IsAbs(baseManifestPath) {
		baseManifestPath = filepath.Join(configDir, baseManifestPath)
	}

	// Read the base manifest file
	data, err := fileReader.ReadFile(baseManifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base manifest file %s: %w", baseManifestPath, err)
	}

	// Unmarshal the YAML into CronWorkflow
	var baseCronWorkflow argoworkflowsv1alpha1.CronWorkflow
	if err := kyaml.Unmarshal(data, &baseCronWorkflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base manifest file %s: %w", baseManifestPath, err)
	}

	return &baseCronWorkflow, nil
}

// GetIndent returns the indent value for YAML generation, defaulting to 2 if not set
func (u *Unit) GetIndent() int {
	if u.Indent == nil {
		return 2 // Default indent is 2 spaces (Kubernetes standard)
	}
	return *u.Indent
}

// ValidateConfig validates the configuration settings
func (c *Config) ValidateConfig(configDir string) error {
	if len(c.Units) == 0 {
		return fmt.Errorf("configuration must contain at least one unit")
	}

	for i, unit := range c.Units {
		if err := unit.Validate(configDir); err != nil {
			return fmt.Errorf("validation failed for unit %d: %w", i, err)
		}
	}

	return nil
}

// Validate validates a single unit configuration
func (u *Unit) Validate(configDir string) error {
	// Check output directory
	if u.OutputDirectory == "" {
		return fmt.Errorf("outputDirectory is required")
	}

	// Resolve output directory path
	outputDir := u.OutputDirectory
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(configDir, outputDir)
	}

	// Check if output directory exists or if its parent exists and is writable
	if info, err := os.Stat(outputDir); err != nil {
		if os.IsNotExist(err) {
			// Check if parent directory exists and is writable
			parentDir := filepath.Dir(outputDir)
			if parentInfo, parentErr := os.Stat(parentDir); parentErr != nil {
				return fmt.Errorf("output directory parent %s does not exist: %w", parentDir, parentErr)
			} else if !parentInfo.IsDir() {
				return fmt.Errorf("output directory parent %s is not a directory", parentDir)
			}
			// Try to create the output directory to verify writability
			if createErr := os.MkdirAll(outputDir, 0755); createErr != nil {
				return fmt.Errorf("cannot create output directory %s: %w", outputDir, createErr)
			}
		} else {
			return fmt.Errorf("cannot access output directory %s: %w", outputDir, err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("output directory %s exists but is not a directory", outputDir)
	}

	// Check base manifest path if provided
	if u.BaseManifestPath != nil {
		baseManifestPath := *u.BaseManifestPath
		if !filepath.IsAbs(baseManifestPath) {
			baseManifestPath = filepath.Join(configDir, baseManifestPath)
		}

		if _, err := os.Stat(baseManifestPath); err != nil {
			return fmt.Errorf("baseManifestPath %s does not exist or cannot be accessed: %w", baseManifestPath, err)
		}
	}

	// Validate indent if provided
	if u.Indent != nil {
		if *u.Indent < 1 || *u.Indent > 8 {
			return fmt.Errorf("indent must be between 1 and 8, got %d", *u.Indent)
		}
	}

	// Check that we have at least one value
	if len(u.Values) == 0 {
		return fmt.Errorf("unit must contain at least one value")
	}

	// Validate each value
	for i, value := range u.Values {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("validation failed for value %d (%s): %w", i, value.Filename, err)
		}
	}

	return nil
}

// Validate validates a single value configuration
func (v *Value) Validate() error {
	if v.Filename == "" {
		return fmt.Errorf("filename is required")
	}

	// Validate each path value
	for i, pv := range v.Paths {
		if err := pv.Validate(); err != nil {
			return fmt.Errorf("validation failed for path %d: %w", i, err)
		}
	}

	return nil
}

// Validate validates a single path-value pair
func (pv *PathValue) Validate() error {
	if pv.Path == "" {
		return fmt.Errorf("path is required")
	}

	// JSONPath expression should start with '$'
	if len(pv.Path) == 0 || pv.Path[0] != '$' {
		return fmt.Errorf("path must be a valid JSONPath expression starting with '$', got: %s", pv.Path)
	}

	// Value can be empty string, so no validation needed for Value field
	return nil
}
