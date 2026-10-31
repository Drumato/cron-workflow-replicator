package config

import (
	"fmt"
	"slices"
)

// NormalizeLegacySchedule moves the singular spec.schedule field, which was removed
// in Argo Workflows v4, into spec.schedules so that manifests written for v3 keep working.
//
// When overwrite is true, the legacy value replaces spec.schedules entirely; this is used
// when a value explicitly sets $.spec.schedule, which overrode the base schedule before.
// Otherwise it is prepended to spec.schedules unless already present, the same way
// Argo Workflows v3.6+ combines both fields.
//
// It reports whether obj contained spec.schedule.
func NormalizeLegacySchedule(obj map[string]any, overwrite bool) (bool, error) {
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return false, nil
	}
	legacy, ok := spec["schedule"]
	if !ok {
		return false, nil
	}
	delete(spec, "schedule")

	schedule, ok := legacy.(string)
	if !ok {
		return true, fmt.Errorf("spec.schedule must be a string, got %T", legacy)
	}
	if schedule == "" {
		return true, nil
	}

	if overwrite {
		spec["schedules"] = []any{schedule}
		return true, nil
	}

	var schedules []any
	if existing, ok := spec["schedules"]; ok && existing != nil {
		schedules, ok = existing.([]any)
		if !ok {
			return true, fmt.Errorf("spec.schedules must be a list, got %T", existing)
		}
	}
	if !slices.Contains(schedules, any(schedule)) {
		schedules = append([]any{schedule}, schedules...)
	}
	spec["schedules"] = schedules
	return true, nil
}
