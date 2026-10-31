package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLegacySchedule(t *testing.T) {
	tests := []struct {
		name         string
		obj          map[string]any
		overwrite    bool
		wantMigrated bool
		wantSpec     map[string]any
		wantErr      bool
	}{
		{
			name:         "no spec",
			obj:          map[string]any{"kind": "CronWorkflow"},
			wantMigrated: false,
		},
		{
			name:         "no legacy schedule",
			obj:          map[string]any{"spec": map[string]any{"schedules": []any{"0 0 * * *"}}},
			wantMigrated: false,
			wantSpec:     map[string]any{"schedules": []any{"0 0 * * *"}},
		},
		{
			name:         "schedule only",
			obj:          map[string]any{"spec": map[string]any{"schedule": "0 0 * * *"}},
			wantMigrated: true,
			wantSpec:     map[string]any{"schedules": []any{"0 0 * * *"}},
		},
		{
			name: "schedule is prepended to schedules",
			obj: map[string]any{"spec": map[string]any{
				"schedule":  "0 0 * * *",
				"schedules": []any{"0 12 * * *"},
			}},
			wantMigrated: true,
			wantSpec:     map[string]any{"schedules": []any{"0 0 * * *", "0 12 * * *"}},
		},
		{
			name: "schedule already in schedules is not duplicated",
			obj: map[string]any{"spec": map[string]any{
				"schedule":  "0 0 * * *",
				"schedules": []any{"0 0 * * *"},
			}},
			wantMigrated: true,
			wantSpec:     map[string]any{"schedules": []any{"0 0 * * *"}},
		},
		{
			name: "overwrite replaces schedules",
			obj: map[string]any{"spec": map[string]any{
				"schedule":  "0 6 * * *",
				"schedules": []any{"0 0 * * *", "0 12 * * *"},
			}},
			overwrite:    true,
			wantMigrated: true,
			wantSpec:     map[string]any{"schedules": []any{"0 6 * * *"}},
		},
		{
			name:         "empty schedule is dropped",
			obj:          map[string]any{"spec": map[string]any{"schedule": ""}},
			wantMigrated: true,
			wantSpec:     map[string]any{},
		},
		{
			name:         "non-string schedule",
			obj:          map[string]any{"spec": map[string]any{"schedule": []any{"0 0 * * *"}}},
			wantMigrated: true,
			wantErr:      true,
		},
		{
			name: "non-list schedules",
			obj: map[string]any{"spec": map[string]any{
				"schedule":  "0 0 * * *",
				"schedules": "0 12 * * *",
			}},
			wantMigrated: true,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrated, err := NormalizeLegacySchedule(tt.obj, tt.overwrite)
			assert.Equal(t, tt.wantMigrated, migrated)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantSpec != nil {
				assert.Equal(t, tt.wantSpec, tt.obj["spec"])
			}
		})
	}
}
