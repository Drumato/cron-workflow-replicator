package jsonpath

import (
	"encoding/json"
	"fmt"
	"log/slog"

	argoworkflowsv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/drumato/cron-workflow-replicator/config"
	"github.com/oliveagle/jsonpath"
)

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
	// Try boolean conversion first
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Return as string if no conversion needed
	return value
}

// setValueAtPath sets a value at the specified JSONPath in the target map
func (pe *PathEvaluator) setValueAtPath(target map[string]interface{}, path string, value string) error {
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
	// This is a simplified implementation - in a production system you'd want
	// more sophisticated path creation logic

	// For now, we'll use a different approach: try to set the value directly
	// using string manipulation to build the nested structure

	// Parse the JSONPath to understand the structure
	segments, err := pe.parseJSONPath(path)
	if err != nil {
		return fmt.Errorf("failed to parse JSONPath: %w", err)
	}

	// Build nested structure
	current := target
	for i, segment := range segments {
		if i == len(segments)-1 {
			// Last segment, set the value with type conversion
			current[segment] = pe.convertValue(value)
		} else {
			// Intermediate segment, create map if it doesn't exist
			if _, exists := current[segment]; !exists {
				current[segment] = make(map[string]interface{})
			}

			// Move to next level
			if nextMap, ok := current[segment].(map[string]interface{}); ok {
				current = nextMap
			} else {
				return fmt.Errorf("path segment %s is not a map, cannot create nested structure", segment)
			}
		}
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
			// Last segment, set the value with type conversion
			current[segment] = pe.convertValue(value)
			return nil
		}

		// Navigate to next level
		if nextMap, ok := current[segment].(map[string]interface{}); ok {
			current = nextMap
		} else {
			return fmt.Errorf("path segment %s is not a map, cannot navigate", segment)
		}
	}

	return nil
}

// parseJSONPath parses a JSONPath expression into segments
// This is a simplified parser that handles basic dot notation
func (pe *PathEvaluator) parseJSONPath(path string) ([]string, error) {
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
		return []string{}, nil
	}

	// Split by dots
	segments := []string{}
	current := ""

	for i, char := range path {
		if char == '.' {
			if current != "" {
				segments = append(segments, current)
				current = ""
			}
		} else {
			current += string(char)
		}

		// Handle last segment
		if i == len(path)-1 && current != "" {
			segments = append(segments, current)
		}
	}

	return segments, nil
}
