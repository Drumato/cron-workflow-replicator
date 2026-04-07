package types

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"gopkg.in/yaml.v3"
	k8syaml "sigs.k8s.io/yaml"
)

// pointerStructFieldNames は CronWorkflowSpec 配下で「ポインタ struct 型」として
// 定義されているフィールドの JSON キー名集合。
//
// Argo 公式型 (argoworkflowsv1alpha1.CronWorkflowSpec) を reflect で walk して
// 自動収集する。これらのフィールドは k8syaml.Marshal の際に omitempty が効くため、
// YAML に出現する時点で必ず non-nil ポインタとなっている。すなわち「ユーザーが
// 明示的に存在させた」フィールドであり、中身が空構造体 ({}) であっても保持すべき
// ディスクリミネータ的フィールドとみなす (ArchiveStrategy.Tar など)。
var pointerStructFieldNames = collectPointerStructFieldNames(
	reflect.TypeOf(argoworkflowsv1alpha1.CronWorkflowSpec{}),
)

// collectPointerStructFieldNames は root から再帰的に型を辿り、
// ポインタ struct 型フィールドの JSON 名を収集します。
func collectPointerStructFieldNames(root reflect.Type) map[string]struct{} {
	names := make(map[string]struct{})
	visited := make(map[reflect.Type]bool)

	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			walk(t.Elem())
			return
		case reflect.Struct:
			// fall through
		default:
			return
		}
		if visited[t] {
			return
		}
		visited[t] = true
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			ft := f.Type
			if ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct {
				if name := jsonFieldName(f); name != "" {
					names[name] = struct{}{}
				}
			}
			walk(ft)
		}
	}
	walk(root)
	return names
}

// jsonFieldName は struct field の json タグからフィールド名を取り出します。
func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	if i := strings.Index(tag, ","); i >= 0 {
		return tag[:i]
	}
	return tag
}

// CleanCronWorkflow - YAML出力用の不要フィールドを除いたCronWorkflow表現
type CleanCronWorkflow struct {
	APIVersion string                                 `yaml:"apiVersion"`
	Kind       string                                 `yaml:"kind"`
	Metadata   *CleanObjectMeta                       `yaml:"metadata,omitempty"`
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
	data := make(map[string]any)

	data["apiVersion"] = c.APIVersion
	data["kind"] = c.Kind

	if c.Metadata != nil {
		metadata := make(map[string]any)

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

	var specMap map[string]any
	if err := yaml.Unmarshal(specData, &specMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
	}

	// 空フィールドを除外
	cleanSpecInterface := removeEmptyFields(specMap)

	// 型アサーションでmap[string]interface{}に戻す
	if cleanSpecMap, ok := cleanSpecInterface.(map[string]any); ok && len(cleanSpecMap) > 0 {
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

// removeEmptyFields は空のフィールドを再帰的に除外します。
// ただし pointerStructFieldNames に含まれるキー (Argo 型でポインタ struct
// として定義されているフィールド) は、空マップであってもユーザーが明示的に
// 設定したものとして保持します (例: archive.tar = {})。
func removeEmptyFields(data any) any {
	return removeEmptyFieldsCtx(data, "")
}

func removeEmptyFieldsCtx(data any, currentKey string) any {
	switch v := data.(type) {
	case map[string]any:
		// 既に空マップで、かつそのキーが保持対象なら、空マップのまま返す。
		// 呼び出し側で「保持対象キー」として再度判定するため、ここでの返却値は
		// nil ではなく空マップである必要がある。
		if len(v) == 0 {
			return map[string]any{}
		}
		result := make(map[string]any)
		for key, value := range v {
			cleaned := removeEmptyFieldsCtx(value, key)
			if _, keep := pointerStructFieldNames[key]; keep {
				// ポインタ struct 由来のフィールドは空でも保持
				if cleaned == nil {
					cleaned = map[string]any{}
				}
				result[key] = cleaned
				continue
			}
			if !isEmpty(cleaned) {
				result[key] = cleaned
			}
		}
		return result
	case []any:
		var result []any
		for _, item := range v {
			cleaned := removeEmptyFieldsCtx(item, currentKey)
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
func isEmpty(value any) bool {
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
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}
