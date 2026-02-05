package jsonpath

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/drumato/cron-workflow-replicator/config"
	"github.com/oliveagle/jsonpath"
)

// PathSegment represents a segment in a JSONPath
type PathSegment struct {
	Key        string
	ArrayIndex *int // nil if not an array access
	IsNegative bool // true for negative indices like [-1]
}

// PathEvaluator handles JSONPath evaluation and value setting
type PathEvaluator struct {
	logger *slog.Logger
}

// NewPathEvaluator creates a new PathEvaluator instance
func NewPathEvaluator(logger *slog.Logger) *PathEvaluator {
	return &PathEvaluator{
		logger: logger,
	}
}

// ApplyPaths applies all path-value pairs to the target CronWorkflow
func (pe *PathEvaluator) ApplyPaths(target *argoworkflowsv1alpha1.CronWorkflow, paths []config.PathValue) error {
	if len(paths) == 0 {
		return nil // Nothing to apply
	}

	// Convert CronWorkflow to map[string]interface{} for JSONPath operations
	targetMap, err := pe.structToMap(target)
	if err != nil {
		return fmt.Errorf("failed to convert CronWorkflow to map: %w", err)
	}

	// Apply each path-value pair
	for _, pv := range paths {
		if err := pe.setValueAtPath(targetMap, pv.Path, pv.Value); err != nil {
			return fmt.Errorf("failed to apply path %s: %w", pv.Path, err)
		}
		pe.logger.Debug("Applied path", "path", pv.Path, "value", pv.Value)
	}

	// Convert back to CronWorkflow
	if err := pe.mapToStruct(targetMap, target); err != nil {
		return fmt.Errorf("failed to convert map back to CronWorkflow: %w", err)
	}

	return nil
}

// structToMap converts a struct to map[string]interface{} via JSON marshaling
func (pe *PathEvaluator) structToMap(obj interface{}) (map[string]interface{}, error) {
	// Marshal to JSON
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	// Unmarshal to map
	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	return result, nil
}

// mapToStruct converts map[string]interface{} back to a struct via JSON marshaling
func (pe *PathEvaluator) mapToStruct(m map[string]interface{}, target interface{}) error {
	// Marshal to JSON
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal map to JSON: %w", err)
	}

	// Unmarshal to target struct
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal to struct: %w", err)
	}

	return nil
}

// convertValue attempts to convert a string value to the appropriate type
func (pe *PathEvaluator) convertValue(value string) interface{} {
	// Try null conversion first
	if value == "null" {
		return nil
	}

	// Try boolean conversion
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Try integer conversion
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Try float conversion
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Try JSON array conversion
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		var arrayVal []interface{}
		if err := json.Unmarshal([]byte(value), &arrayVal); err == nil {
			return arrayVal
		}
	}

	// Try JSON object conversion
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		var objVal map[string]interface{}
		if err := json.Unmarshal([]byte(value), &objVal); err == nil {
			return objVal
		}
	}

	// Return as string if no conversion needed
	return value
}

// setValueAtPath sets a value at the specified JSONPath in the target map
func (pe *PathEvaluator) setValueAtPath(target map[string]interface{}, path string, value string) error {
	// Convert value first to check if it's an array
	convertedValue := pe.convertValue(value)

	// Check if this is an array value and the path doesn't specify an array element
	if arrayVal, isArray := convertedValue.([]interface{}); isArray {
		if !pe.isArrayElementPath(path) {
			// This is a whole array assignment
			return pe.setArrayValue(target, path, arrayVal)
		}
	}

	// Compile JSONPath
	compiled, err := jsonpath.Compile(path)
	if err != nil {
		return fmt.Errorf("failed to compile JSONPath %s: %w", path, err)
	}

	// Check if the path exists and what it points to
	existingValue, err := compiled.Lookup(target)
	if err != nil {
		// Path doesn't exist, we need to create it
		return pe.createPathAndSetValue(target, path, value)
	}

	// Path exists, replace the value
	return pe.replaceValueAtPath(target, path, value, existingValue)
}

// createPathAndSetValue creates the path structure and sets the value
func (pe *PathEvaluator) createPathAndSetValue(target map[string]interface{}, path string, value string) error {
	// Parse the JSONPath to understand the structure
	segments, err := pe.parseJSONPath(path)
	if err != nil {
		return fmt.Errorf("failed to parse JSONPath: %w", err)
	}

	// Build nested structure
	current := target
	for i, segment := range segments {
		if i == len(segments)-1 {
			// Last segment, set the value
			if segment.ArrayIndex != nil {
				// Array access
				return pe.setValueAtArrayIndex(current, segment.Key, *segment.ArrayIndex, segment.IsNegative, value)
			} else {
				// Regular key access
				current[segment.Key] = pe.convertValue(value)
			}
		} else {
			// Intermediate segment
			if segment.ArrayIndex != nil {
				// Navigate through array
				if err := pe.navigateToArrayElement(&current, segment.Key, *segment.ArrayIndex, segment.IsNegative); err != nil {
					return err
				}
			} else {
				// Create map if it doesn't exist
				if _, exists := current[segment.Key]; !exists {
					current[segment.Key] = make(map[string]interface{})
				}

				// Move to next level
				if nextMap, ok := current[segment.Key].(map[string]interface{}); ok {
					current = nextMap
				} else {
					return fmt.Errorf("path segment %s is not a map, cannot create nested structure", segment.Key)
				}
			}
		}
	}

	return nil
}

// setValueAtArrayIndex sets a value at a specific array index
func (pe *PathEvaluator) setValueAtArrayIndex(parent map[string]interface{}, arrayKey string, index int, isNegative bool, value string) error {
	// Get or create the array
	var arr []interface{}
	if existing, exists := parent[arrayKey]; exists {
		if existingArr, ok := existing.([]interface{}); ok {
			arr = existingArr
		} else {
			return fmt.Errorf("key %s exists but is not an array", arrayKey)
		}
	} else {
		arr = make([]interface{}, 0)
	}

	// Calculate actual index
	actualIndex := index
	if isNegative {
		if len(arr) == 0 {
			return fmt.Errorf("cannot use negative index %d on empty array", index)
		}
		actualIndex = len(arr) + index // index is negative, so this is len(arr) - abs(index)
		if actualIndex < 0 {
			return fmt.Errorf("negative index %d is out of bounds for array of length %d", index, len(arr))
		}
	}

	// Extend array if necessary (but only for positive indices)
	if actualIndex >= len(arr) {
		// Extend array with nil values
		for len(arr) <= actualIndex {
			arr = append(arr, nil)
		}
	}

	// Set the value
	arr[actualIndex] = pe.convertValue(value)
	parent[arrayKey] = arr

	return nil
}

// navigateToArrayElement navigates to an array element and updates the current pointer
func (pe *PathEvaluator) navigateToArrayElement(current *map[string]interface{}, arrayKey string, index int, isNegative bool) error {
	// Get or create the array
	var arr []interface{}
	if arrayVal, exists := (*current)[arrayKey]; exists {
		if existingArr, ok := arrayVal.([]interface{}); ok {
			arr = existingArr
		} else {
			return fmt.Errorf("key %s exists but is not an array", arrayKey)
		}
	} else {
		// Create array if it doesn't exist
		arr = make([]interface{}, 0)
		(*current)[arrayKey] = arr
	}

	// Calculate actual index
	actualIndex := index
	if isNegative {
		if len(arr) == 0 {
			return fmt.Errorf("cannot use negative index %d on empty array", index)
		}
		actualIndex = len(arr) + index // index is negative, so this is len(arr) - abs(index)
		if actualIndex < 0 {
			return fmt.Errorf("negative index %d is out of bounds for array of length %d", index, len(arr))
		}
	}

	// Extend array if necessary (but only for positive indices)
	if actualIndex >= len(arr) && !isNegative {
		for len(arr) <= actualIndex {
			arr = append(arr, make(map[string]interface{}))
		}
		(*current)[arrayKey] = arr
	}

	// Check bounds after extension
	if actualIndex >= len(arr) {
		return fmt.Errorf("index %d is out of bounds for array of length %d", actualIndex, len(arr))
	}

	// Navigate to the array element
	element := arr[actualIndex]
	if elementMap, ok := element.(map[string]interface{}); ok {
		*current = elementMap
	} else {
		// Create a map if the element is not already a map
		newMap := make(map[string]interface{})
		arr[actualIndex] = newMap
		(*current)[arrayKey] = arr
		*current = newMap
	}

	return nil
}

// replaceValueAtPath replaces an existing value at the JSONPath
func (pe *PathEvaluator) replaceValueAtPath(target map[string]interface{}, path string, value string, existingValue interface{}) error {
	// Parse the JSONPath to understand the structure
	segments, err := pe.parseJSONPath(path)
	if err != nil {
		return fmt.Errorf("failed to parse JSONPath: %w", err)
	}

	// Navigate to the parent of the target field
	current := target
	for i, segment := range segments {
		if i == len(segments)-1 {
			// Last segment, set the value
			if segment.ArrayIndex != nil {
				// Array access
				return pe.setValueAtArrayIndex(current, segment.Key, *segment.ArrayIndex, segment.IsNegative, value)
			} else {
				// Regular key access
				current[segment.Key] = pe.convertValue(value)
				return nil
			}
		}

		// Navigate to next level
		if segment.ArrayIndex != nil {
			// Navigate through array
			if err := pe.navigateToArrayElement(&current, segment.Key, *segment.ArrayIndex, segment.IsNegative); err != nil {
				return err
			}
		} else {
			if nextMap, ok := current[segment.Key].(map[string]interface{}); ok {
				current = nextMap
			} else {
				return fmt.Errorf("path segment %s is not a map, cannot navigate", segment.Key)
			}
		}
	}

	return nil
}

// isArrayElementPath checks if the JSONPath specifies an array element (e.g., $.path[0] or $.path[0].name)
func (pe *PathEvaluator) isArrayElementPath(path string) bool {
	// Parse the path to check if any segment contains an array index
	segments, err := pe.parseJSONPath(path)
	if err != nil {
		return false
	}

	// Check if any segment has an array index
	for _, segment := range segments {
		if segment.ArrayIndex != nil {
			return true
		}
	}

	return false
}

// setArrayValue sets an entire array at the specified JSONPath
func (pe *PathEvaluator) setArrayValue(target map[string]interface{}, path string, arrayVal []interface{}) error {
	// Parse the JSONPath to understand the structure
	segments, err := pe.parseJSONPath(path)
	if err != nil {
		return fmt.Errorf("failed to parse JSONPath: %w", err)
	}

	// Navigate to the parent of the target field
	current := target
	for i, segment := range segments {
		if i == len(segments)-1 {
			// Last segment, set the array value
			current[segment.Key] = arrayVal
			return nil
		}

		// Navigate to next level
		if segment.ArrayIndex != nil {
			// Navigate through array
			if err := pe.navigateToArrayElement(&current, segment.Key, *segment.ArrayIndex, segment.IsNegative); err != nil {
				return err
			}
		} else {
			// Create map if it doesn't exist
			if _, exists := current[segment.Key]; !exists {
				current[segment.Key] = make(map[string]interface{})
			}

			// Move to next level
			if nextMap, ok := current[segment.Key].(map[string]interface{}); ok {
				current = nextMap
			} else {
				return fmt.Errorf("path segment %s is not a map, cannot navigate", segment.Key)
			}
		}
	}

	return nil
}

// parseJSONPath parses a JSONPath expression into segments with array index support
func (pe *PathEvaluator) parseJSONPath(path string) ([]PathSegment, error) {
	if len(path) < 1 || path[0] != '$' {
		return nil, fmt.Errorf("invalid JSONPath: must start with '$'")
	}

	// Remove the leading '$'
	path = path[1:]

	// If there's a leading '.', remove it
	if len(path) > 0 && path[0] == '.' {
		path = path[1:]
	}

	// If path is empty after removing '$' (and potentially '.'), return empty slice
	if path == "" {
		return []PathSegment{}, nil
	}

	// Regular expressions for parsing
	arrayIndexPattern := regexp.MustCompile(`^(.+)\[(-?\d+)\]$`)

	// Split by dots and parse each segment
	segments := []PathSegment{}
	rawSegments := strings.Split(path, ".")

	for _, rawSegment := range rawSegments {
		if rawSegment == "" {
			continue
		}

		// Check if this segment has an array index
		if matches := arrayIndexPattern.FindStringSubmatch(rawSegment); matches != nil {
			key := matches[1]
			indexStr := matches[2]

			// Parse the index
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array index in segment '%s': %w", rawSegment, err)
			}

			isNegative := index < 0

			segments = append(segments, PathSegment{
				Key:        key,
				ArrayIndex: &index,
				IsNegative: isNegative,
			})
		} else {
			// Regular segment without array index
			segments = append(segments, PathSegment{
				Key:        rawSegment,
				ArrayIndex: nil,
				IsNegative: false,
			})
		}
	}

	return segments, nil
}
