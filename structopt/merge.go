package structopt

import (
	"reflect"
)

// MergeStruct merges fields from src into dst. Only non-zero fields from src are copied to dst.
// Both dst and src must be pointers to structs of the same type.
func MergeStruct[T any](dst, src *T) {
	if dst == nil || src == nil {
		return
	}

	dstValue := reflect.ValueOf(dst).Elem()
	srcValue := reflect.ValueOf(src).Elem()

	mergeFields(dstValue, srcValue)
}

// mergeFields recursively merges fields from src to dst
func mergeFields(dst, src reflect.Value) {
	if dst.Type() != src.Type() {
		return
	}

	switch dst.Kind() {
	case reflect.Struct:
		for i := 0; i < dst.NumField(); i++ {
			dstField := dst.Field(i)
			srcField := src.Field(i)

			if !dstField.CanSet() {
				continue
			}

			// If the source field is the zero value, skip it
			if isZeroValue(srcField) {
				continue
			}

			// If the destination field is also a struct, recursively merge
			if dstField.Kind() == reflect.Struct && srcField.Kind() == reflect.Struct {
				mergeFields(dstField, srcField)
			} else if dstField.Kind() == reflect.Map && srcField.Kind() == reflect.Map {
				// For map fields, use special map merging logic
				mergeFields(dstField, srcField)
			} else if dstField.Kind() == reflect.Slice && srcField.Kind() == reflect.Slice {
				// For slice fields, use special slice merging logic
				mergeFields(dstField, srcField)
			} else if dstField.Kind() == reflect.Ptr && srcField.Kind() == reflect.Ptr {
				// For pointer fields, use special pointer merging logic
				mergeFields(dstField, srcField)
			} else {
				// For basic types, copy the value directly
				dstField.Set(srcField)
			}
		}

	case reflect.Ptr:
		if src.IsNil() {
			return
		}

		// If dst is nil, create a new instance
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}

		mergeFields(dst.Elem(), src.Elem())

	case reflect.Slice:
		if src.IsNil() || src.Len() == 0 {
			return
		}
		// For slices, replace the entire slice
		dst.Set(src)

	case reflect.Map:
		if src.IsNil() || src.Len() == 0 {
			return
		}
		// For maps, merge keys
		if dst.IsNil() {
			dst.Set(reflect.MakeMap(dst.Type()))
		}

		for _, key := range src.MapKeys() {
			dst.SetMapIndex(key, src.MapIndex(key))
		}

	default:
		// For basic types, copy the value
		dst.Set(src)
	}
}

// isZeroValue checks if a reflect.Value represents the zero value for its type
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0.0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0.0
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isZeroValue(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isZeroValue(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}