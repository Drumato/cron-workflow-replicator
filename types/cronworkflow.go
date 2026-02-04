package types

import (
	"bytes"
	"fmt"
	"reflect"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	k8syaml "sigs.k8s.io/yaml"
	"gopkg.in/yaml.v3"
)

// CleanCronWorkflow - YAML出力用の不要フィールドを除いたCronWorkflow表現
type CleanCronWorkflow struct {
	APIVersion string                               `yaml:"apiVersion"`
	Kind       string                               `yaml:"kind"`
	Metadata   *CleanObjectMeta                     `yaml:"metadata,omitempty"`
	Spec       argoworkflowsv1alpha1.CronWorkflowSpec `yaml:"spec,omitempty"`
}

// CleanObjectMeta - creationTimestampを除いたObjectMeta表現
type CleanObjectMeta struct {
	Name        string            `yaml:"name,omitempty"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// NewCleanCronWorkflow は既存のCronWorkflowからCleanCronWorkflowを作成します
func NewCleanCronWorkflow(cw *argoworkflowsv1alpha1.CronWorkflow) *CleanCronWorkflow {
	clean := &CleanCronWorkflow{
		APIVersion: cw.APIVersion,
		Kind:       cw.Kind,
		Spec:       cw.Spec,
	}

	// ObjectMetaから必要なフィールドのみをコピー（空でない場合のみMetadataを設定）
	if cw.Name != "" || cw.Namespace != "" ||
		len(cw.Labels) > 0 || len(cw.Annotations) > 0 {
		clean.Metadata = &CleanObjectMeta{
			Name:        cw.Name,
			Namespace:   cw.Namespace,
			Labels:      cw.Labels,
			Annotations: cw.Annotations,
		}
	}

	return clean
}

// ToYAML はCleanCronWorkflowをYAMLバイト列に変換します（デフォルトは2スペースインデント）
func (c *CleanCronWorkflow) ToYAML() ([]byte, error) {
	return c.ToYAMLWithIndent(2)
}

// ToYAMLWithIndent はCleanCronWorkflowを指定されたインデントでYAMLバイト列に変換します
func (c *CleanCronWorkflow) ToYAMLWithIndent(indent int) ([]byte, error) {
	// カスタムマップを作成して正しいキー名にする
	data := make(map[string]interface{})

	data["apiVersion"] = c.APIVersion
	data["kind"] = c.Kind

	if c.Metadata != nil {
		metadata := make(map[string]interface{})

		if c.Metadata.Name != "" {
			metadata["name"] = c.Metadata.Name
		}
		if c.Metadata.Namespace != "" {
			metadata["namespace"] = c.Metadata.Namespace
		}
		if len(c.Metadata.Labels) > 0 {
			metadata["labels"] = c.Metadata.Labels
		}
		if len(c.Metadata.Annotations) > 0 {
			metadata["annotations"] = c.Metadata.Annotations
		}

		if len(metadata) > 0 {
			data["metadata"] = metadata
		}
	}

	// Specをマーシャルしてキャメルケースキーを保持し、空フィールドを除外
	specData, err := k8syaml.Marshal(c.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	var specMap map[string]interface{}
	if err := yaml.Unmarshal(specData, &specMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
	}

	// 空フィールドを除外
	cleanSpecInterface := removeEmptyFields(specMap)

	// 型アサーションでmap[string]interface{}に戻す
	if cleanSpecMap, ok := cleanSpecInterface.(map[string]interface{}); ok && len(cleanSpecMap) > 0 {
		data["spec"] = cleanSpecMap
	}

	// カスタムインデントでYAMLを生成
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(indent)

	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to encode YAML with custom indent: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to close YAML encoder: %w", err)
	}

	return buf.Bytes(), nil
}

// removeEmptyFields は空のフィールドを再帰的に除外します
func removeEmptyFields(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			cleaned := removeEmptyFields(value)
			if !isEmpty(cleaned) {
				result[key] = cleaned
			}
		}
		return result
	case []interface{}:
		var result []interface{}
		for _, item := range v {
			cleaned := removeEmptyFields(item)
			if !isEmpty(cleaned) {
				result = append(result, cleaned)
			}
		}
		return result
	default:
		return v
	}
}

// isEmpty は値が空かどうかを判定します
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Array, reflect.Map:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}