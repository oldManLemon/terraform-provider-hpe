// (C) Copyright 2026 Hewlett Packard Enterprise Development LP

package instance

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func tagSet(pairs ...string) types.Set {
	elems := make([]attr.Value, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		elems = append(elems, TagsValue{
			Name:  types.StringValue(pairs[i]),
			Value: types.StringValue(pairs[i+1]),
			state: attr.ValueStateKnown,
		})
	}

	return types.SetValueMust(TagsValue{}.Type(context.Background()), elems)
}

func nullTags() types.Set {
	return types.SetNull(TagsValue{}.Type(context.Background()))
}

// TestUnitReconcileTags verifies that a post-apply read keeps the configured
// tags, with a warning, when Morpheus reports a different set (tags added by a
// policy or workflow task, or none at all), so Create and Update no longer fail
// with "Provider produced inconsistent result after apply". A plain refresh
// must still store the API value so the difference is detected as drift, and a
// configured empty set must stay empty rather than become null.
func TestUnitReconcileTags(t *testing.T) {
	t.Parallel()

	none := nullTags()
	empty := tagSet()
	backup := tagSet("backup", "daily")
	prod := tagSet("env", "prod")
	prodAndBackup := tagSet("env", "prod", "backup", "daily")

	tests := []struct {
		name     string
		plan     types.Set
		api      types.Set
		refresh  bool
		want     types.Set
		warnings int
	}{
		{"not configured uses the API value", none, backup, false, backup, 0},
		{"empty set stays empty after apply", empty, none, false, empty, 0},
		{"empty set stays empty on refresh", empty, none, true, empty, 0},
		{"API matches the plan", prod, prod, false, prod, 0},
		{"API dropped the tags after apply", prod, none, false, prod, 1},
		{"API added a tag after apply", prod, prodAndBackup, false, prod, 1},
		{"API dropped the tags on refresh", prod, none, true, none, 0},
		{"API added a tag on refresh", prod, prodAndBackup, true, prodAndBackup, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, diags := reconcileTags(42, tt.plan, tt.api, tt.refresh)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags.Errors())
			}

			if !got.Equal(tt.want) {
				t.Errorf("reconcileTags() = %s, want %s", got, tt.want)
			}

			if n := diags.WarningsCount(); n != tt.warnings {
				t.Errorf("reconcileTags() raised %d warnings, want %d", n, tt.warnings)
			}
		})
	}
}
