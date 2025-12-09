package structutil

import (
	"reflect"
	"testing"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Test structs for basic functionality
type TestStruct struct {
	Name         string
	Age          int
	Active       bool
	Score        float64
	Tags         []string
	Labels       map[string]string
	NestedStruct NestedStruct
	Pointer      *string
}

type NestedStruct struct {
	Field1 string
	Field2 int
}

func TestMergeStruct_BasicFields(t *testing.T) {
	tests := []struct {
		name     string
		dst      TestStruct
		src      TestStruct
		expected TestStruct
	}{
		{
			name: "merge non-zero fields into empty struct",
			dst:  TestStruct{},
			src: TestStruct{
				Name:   "test",
				Age:    25,
				Active: true,
				Score:  95.5,
			},
			expected: TestStruct{
				Name:   "test",
				Age:    25,
				Active: true,
				Score:  95.5,
			},
		},
		{
			name: "merge into struct with existing values - no overwrite of zero values",
			dst: TestStruct{
				Name: "existing",
				Age:  30,
			},
			src: TestStruct{
				Age:    0, // zero value, should not overwrite
				Active: true,
				Score:  88.0,
			},
			expected: TestStruct{
				Name:   "existing",
				Age:    30, // should keep existing value since src is zero
				Active: true,
				Score:  88.0,
			},
		},
		{
			name: "merge overwrites existing non-zero values",
			dst: TestStruct{
				Name: "existing",
				Age:  30,
			},
			src: TestStruct{
				Name:   "new",
				Active: true,
			},
			expected: TestStruct{
				Name:   "new", // should overwrite
				Age:    30,
				Active: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MergeStruct(&tt.dst, &tt.src)
			assert.Equal(t, tt.expected, tt.dst)
		})
	}
}

func TestMergeStruct_SlicesAndMaps(t *testing.T) {
	tests := []struct {
		name     string
		dst      TestStruct
		src      TestStruct
		expected TestStruct
	}{
		{
			name: "merge slice replaces entirely",
			dst: TestStruct{
				Tags: []string{"old"},
			},
			src: TestStruct{
				Tags: []string{"new1", "new2"},
			},
			expected: TestStruct{
				Tags: []string{"new1", "new2"},
			},
		},
		{
			name: "merge map adds keys",
			dst: TestStruct{
				Labels: map[string]string{
					"existing": "value",
				},
			},
			src: TestStruct{
				Labels: map[string]string{
					"new": "newvalue",
				},
			},
			expected: TestStruct{
				Labels: map[string]string{
					"existing": "value",
					"new":      "newvalue",
				},
			},
		},
		{
			name: "empty slices and maps are ignored",
			dst: TestStruct{
				Tags: []string{"keep"},
				Labels: map[string]string{
					"keep": "this",
				},
			},
			src: TestStruct{
				Tags:   []string{},          // empty slice should be ignored
				Labels: map[string]string{}, // empty map should be ignored
			},
			expected: TestStruct{
				Tags: []string{"keep"},
				Labels: map[string]string{
					"keep": "this",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MergeStruct(&tt.dst, &tt.src)
			assert.Equal(t, tt.expected, tt.dst)
		})
	}
}

func TestMergeStruct_NestedStructs(t *testing.T) {
	tests := []struct {
		name     string
		dst      TestStruct
		src      TestStruct
		expected TestStruct
	}{
		{
			name: "merge nested struct fields",
			dst: TestStruct{
				NestedStruct: NestedStruct{
					Field1: "existing",
				},
			},
			src: TestStruct{
				NestedStruct: NestedStruct{
					Field2: 42,
				},
			},
			expected: TestStruct{
				NestedStruct: NestedStruct{
					Field1: "existing",
					Field2: 42,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MergeStruct(&tt.dst, &tt.src)
			assert.Equal(t, tt.expected, tt.dst)
		})
	}
}

func TestMergeStruct_Pointers(t *testing.T) {
	str1 := "original"
	str2 := "new"

	tests := []struct {
		name     string
		dst      TestStruct
		src      TestStruct
		expected TestStruct
	}{
		{
			name: "merge into nil pointer",
			dst:  TestStruct{},
			src: TestStruct{
				Pointer: &str2,
			},
			expected: TestStruct{
				Pointer: &str2,
			},
		},
		{
			name: "merge into existing pointer",
			dst: TestStruct{
				Pointer: &str1,
			},
			src: TestStruct{
				Pointer: &str2,
			},
			expected: TestStruct{
				Pointer: &str2,
			},
		},
		{
			name: "nil pointer in source is ignored",
			dst: TestStruct{
				Pointer: &str1,
			},
			src: TestStruct{
				Pointer: nil,
			},
			expected: TestStruct{
				Pointer: &str1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MergeStruct(&tt.dst, &tt.src)
			if tt.expected.Pointer == nil {
				assert.Nil(t, tt.dst.Pointer)
			} else {
				assert.NotNil(t, tt.dst.Pointer)
				assert.Equal(t, *tt.expected.Pointer, *tt.dst.Pointer)
			}
		})
	}
}

func TestMergeStruct_CronWorkflow(t *testing.T) {
	// Test with actual CronWorkflow types that are used in the application
	tests := []struct {
		name     string
		dst      argoworkflowsv1alpha1.CronWorkflow
		srcMeta  metav1.ObjectMeta
		srcSpec  argoworkflowsv1alpha1.CronWorkflowSpec
		expected argoworkflowsv1alpha1.CronWorkflow
	}{
		{
			name: "merge metadata and spec into empty CronWorkflow",
			dst: argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
			},
			srcMeta: metav1.ObjectMeta{
				Name:      "test-workflow",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
			},
			srcSpec: argoworkflowsv1alpha1.CronWorkflowSpec{
				Schedule: "0 0 * * *",
				Timezone: "UTC",
			},
			expected: argoworkflowsv1alpha1.CronWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "CronWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: argoworkflowsv1alpha1.CronWorkflowSpec{
					Schedule: "0 0 * * *",
					Timezone: "UTC",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Merge metadata
			MergeStruct(&tt.dst.ObjectMeta, &tt.srcMeta)
			// Merge spec
			MergeStruct(&tt.dst.Spec, &tt.srcSpec)

			assert.Equal(t, tt.expected, tt.dst)
		})
	}
}

func TestMergeStruct_NilInputs(t *testing.T) {
	// Test nil safety
	var dst *TestStruct
	src := &TestStruct{Name: "test"}

	// Should not panic
	MergeStruct(dst, src)

	dst = &TestStruct{Name: "existing"}
	var src2 *TestStruct

	// Should not panic
	MergeStruct(dst, src2)

	// dst should be unchanged
	assert.Equal(t, "existing", dst.Name)
}

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"zero string", "", true},
		{"non-zero string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"zero bool", false, true},
		{"non-zero bool", true, false},
		{"zero float", 0.0, true},
		{"non-zero float", 3.14, false},
		{"nil slice", []string(nil), true},
		{"empty slice", []string{}, true},
		{"non-empty slice", []string{"item"}, false},
		{"nil map", map[string]string(nil), true},
		{"empty map", map[string]string{}, true},
		{"non-empty map", map[string]string{"key": "value"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			result := isZeroValue(v)
			assert.Equal(t, tt.expected, result)
		})
	}
}
