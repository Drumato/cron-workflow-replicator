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

// FilterExpression represents a filter condition in a JSONPath (e.g., [?(@.name == 'task')])
type FilterExpression struct {
	Key   string // filter target key (e.g., "name")
	Value string // value to match (e.g., "task")
}

// PathSegment represents a segment in a JSONPath
type PathSegment struct {
	Key        string
	ArrayIndex *int              // nil if not an array access
	IsNegative bool              // true for negative indices like [-1]
	Filter     *FilterExpression // nil if not a filter expression
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
func (pe *PathEvaluator) structToMap(obj any) (map[string]any, error) {
	// Marshal to JSON
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	// Unmarshal to map
	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	return result, nil
}

// mapToStruct converts map[string]interface{} back to a struct via JSON marshaling
func (pe *PathEvaluator) mapToStruct(m map[string]any, target any) error {
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
func (pe *PathEvaluator) convertValue(value string) any {
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
		var arrayVal []any
		if err := json.Unmarshal([]byte(value), &arrayVal); err == nil {
			return arrayVal
		}
	}

	// Try JSON object conversion
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		var objVal map[string]any
		if err := json.Unmarshal([]byte(value), &objVal); err == nil {
			return objVal
		}
	}

	// Return as string if no conversion needed
	return value
}

// setValueAtPath sets a value at the specified JSONPath in the target map
func (pe *PathEvaluator) setValueAtPath(target map[string]any, path string, value string) error {
	// Convert value first to check if it's an array
	convertedValue := pe.convertValue(value)

	// Check if this is an array value and the path doesn't specify an array element
	if arrayVal, isArray := convertedValue.([]any); isArray {
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
func (pe *PathEvaluator) createPathAndSetValue(target map[string]any, path string, value string) error {
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
			if segment.Filter != nil {
				// Filter-based array access at last segment: need next segment for field key
				// This case means the path ends with a filter, set the whole matched element
				return pe.setValueAtArrayElementByFilter(current, segment.Key, segment.Filter, "", pe.convertValue(value))
			} else if segment.ArrayIndex != nil {
				// Array access
				return pe.setValueAtArrayIndex(current, segment.Key, *segment.ArrayIndex, segment.IsNegative, value)
			} else {
				// Regular key access
				current[segment.Key] = pe.convertValue(value)
			}
		} else {
			// Intermediate segment
			if segment.Filter != nil {
				// Navigate through array by filter
				if err := pe.navigateToArrayElementByFilter(&current, segment.Key, segment.Filter); err != nil {
					return err
				}
			} else if segment.ArrayIndex != nil {
				// Navigate through array
				if err := pe.navigateToArrayElement(&current, segment.Key, *segment.ArrayIndex, segment.IsNegative); err != nil {
					return err
				}
			} else {
				// Create map if it doesn't exist
				if _, exists := current[segment.Key]; !exists {
					current[segment.Key] = make(map[string]any)
				}

				// Move to next level
				if nextMap, ok := current[segment.Key].(map[string]any); ok {
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
func (pe *PathEvaluator) setValueAtArrayIndex(parent map[string]any, arrayKey string, index int, isNegative bool, value string) error {
	// Get or create the array
	var arr []any
	if existing, exists := parent[arrayKey]; exists {
		if existingArr, ok := existing.([]any); ok {
			arr = existingArr
		} else {
			return fmt.Errorf("key %s exists but is not an array", arrayKey)
		}
	} else {
		arr = make([]any, 0)
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
func (pe *PathEvaluator) navigateToArrayElement(current *map[string]any, arrayKey string, index int, isNegative bool) error {
	// Get or create the array
	var arr []any
	if arrayVal, exists := (*current)[arrayKey]; exists {
		if existingArr, ok := arrayVal.([]any); ok {
			arr = existingArr
		} else {
			return fmt.Errorf("key %s exists but is not an array", arrayKey)
		}
	} else {
		// Create array if it doesn't exist
		arr = make([]any, 0)
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
			arr = append(arr, make(map[string]any))
		}
		(*current)[arrayKey] = arr
	}

	// Check bounds after extension
	if actualIndex >= len(arr) {
		return fmt.Errorf("index %d is out of bounds for array of length %d", actualIndex, len(arr))
	}

	// Navigate to the array element
	element := arr[actualIndex]
	if elementMap, ok := element.(map[string]any); ok {
		*current = elementMap
	} else {
		// Create a map if the element is not already a map
		newMap := make(map[string]any)
		arr[actualIndex] = newMap
		(*current)[arrayKey] = arr
		*current = newMap
	}

	return nil
}

// navigateToArrayElementByFilter navigates to the first array element matching the filter and updates the current pointer
func (pe *PathEvaluator) navigateToArrayElementByFilter(current *map[string]any, arrayKey string, filter *FilterExpression) error {
	arrayVal, exists := (*current)[arrayKey]
	if !exists {
		return fmt.Errorf("key %s does not exist", arrayKey)
	}

	arr, ok := arrayVal.([]any)
	if !ok {
		return fmt.Errorf("key %s is not an array", arrayKey)
	}

	for _, element := range arr {
		elementMap, ok := element.(map[string]any)
		if !ok {
			continue
		}
		if val, exists := elementMap[filter.Key]; exists {
			if fmt.Sprintf("%v", val) == filter.Value {
				*current = elementMap
				return nil
			}
		}
	}

	return fmt.Errorf("no element matching filter [?(@.%s == '%s')] found in array %s", filter.Key, filter.Value, arrayKey)
}

// setValueAtArrayElementByFilter sets a value on the first array element matching the filter
func (pe *PathEvaluator) setValueAtArrayElementByFilter(parent map[string]any, arrayKey string, filter *FilterExpression, fieldKey string, value any) error {
	arrayVal, exists := parent[arrayKey]
	if !exists {
		return fmt.Errorf("key %s does not exist", arrayKey)
	}

	arr, ok := arrayVal.([]any)
	if !ok {
		return fmt.Errorf("key %s is not an array", arrayKey)
	}

	for _, element := range arr {
		elementMap, ok := element.(map[string]any)
		if !ok {
			continue
		}
		if val, exists := elementMap[filter.Key]; exists {
			if fmt.Sprintf("%v", val) == filter.Value {
				elementMap[fieldKey] = value
				return nil
			}
		}
	}

	return fmt.Errorf("no element matching filter [?(@.%s == '%s')] found in array %s", filter.Key, filter.Value, arrayKey)
}

// replaceValueAtPath replaces an existing value at the JSONPath
func (pe *PathEvaluator) replaceValueAtPath(target map[string]any, path string, value string, existingValue any) error {
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
			if segment.Filter != nil {
				return pe.setValueAtArrayElementByFilter(current, segment.Key, segment.Filter, "", pe.convertValue(value))
			} else if segment.ArrayIndex != nil {
				// Array access
				return pe.setValueAtArrayIndex(current, segment.Key, *segment.ArrayIndex, segment.IsNegative, value)
			} else {
				// Regular key access
				current[segment.Key] = pe.convertValue(value)
				return nil
			}
		}

		// Navigate to next level
		if segment.Filter != nil {
			if err := pe.navigateToArrayElementByFilter(&current, segment.Key, segment.Filter); err != nil {
				return err
			}
		} else if segment.ArrayIndex != nil {
			// Navigate through array
			if err := pe.navigateToArrayElement(&current, segment.Key, *segment.ArrayIndex, segment.IsNegative); err != nil {
				return err
			}
		} else {
			if nextMap, ok := current[segment.Key].(map[string]any); ok {
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

	// Check if any segment has an array index or filter expression
	for _, segment := range segments {
		if segment.ArrayIndex != nil || segment.Filter != nil {
			return true
		}
	}

	return false
}

// setArrayValue sets an entire array at the specified JSONPath
func (pe *PathEvaluator) setArrayValue(target map[string]any, path string, arrayVal []any) error {
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
		if segment.Filter != nil {
			if err := pe.navigateToArrayElementByFilter(&current, segment.Key, segment.Filter); err != nil {
				return err
			}
		} else if segment.ArrayIndex != nil {
			// Navigate through array
			if err := pe.navigateToArrayElement(&current, segment.Key, *segment.ArrayIndex, segment.IsNegative); err != nil {
				return err
			}
		} else {
			// Create map if it doesn't exist
			if _, exists := current[segment.Key]; !exists {
				current[segment.Key] = make(map[string]any)
			}

			// Move to next level
			if nextMap, ok := current[segment.Key].(map[string]any); ok {
				current = nextMap
			} else {
				return fmt.Errorf("path segment %s is not a map, cannot navigate", segment.Key)
			}
		}
	}

	return nil
}

// splitPathSegments splits a JSONPath by dots, but ignores dots inside bracket expressions [...]
func splitPathSegments(path string) []string {
	var segments []string
	var current strings.Builder
	depth := 0

	for _, ch := range path {
		switch ch {
		case '[':
			depth++
			current.WriteRune(ch)
		case ']':
			depth--
			current.WriteRune(ch)
		case '.':
			if depth == 0 {
				if current.Len() > 0 {
					segments = append(segments, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
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
	filterPattern := regexp.MustCompile(`^(.+)\[\?\(@\.(\w+)\s*==\s*'([^']+)'\)\]$`)

	// Split by dots, but not dots inside bracket expressions [...]
	segments := []PathSegment{}
	rawSegmentsList := splitPathSegments(path)

	for _, rawSegment := range rawSegmentsList {
		if rawSegment == "" {
			continue
		}

		// Check if this segment has a filter expression
		if matches := filterPattern.FindStringSubmatch(rawSegment); matches != nil {
			key := matches[1]
			filterKey := matches[2]
			filterValue := matches[3]

			segments = append(segments, PathSegment{
				Key: key,
				Filter: &FilterExpression{
					Key:   filterKey,
					Value: filterValue,
				},
			})
		} else if matches := arrayIndexPattern.FindStringSubmatch(rawSegment); matches != nil {
			// Check if this segment has an array index
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
