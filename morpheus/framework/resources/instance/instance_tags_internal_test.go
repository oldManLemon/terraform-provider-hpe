// (C) Copyright 2026 Hewlett Packard Enterprise Development LP

package instance

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func tagSet(t *testing.T, pairs ...string) types.Set {
	t.Helper()

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

func countWarnings(diags diag.Diagnostics) int {
	n := 0
	for _, d := range diags {
		if d.Severity() == diag.SeverityWarning {
			n++
		}
	}

	return n
}

// TestUnitReconcileTags covers the cases that made Create/Update fail with
// "Provider produced inconsistent result after apply" on .tags.
func TestUnitReconcileTags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		plan     types.Set
		api      types.Set
		refresh  bool
		want     types.Set
		warnings int
	}{
		{
			name: "tags not configured, API value is used",
			plan: nullTags(),
			api:  tagSet(t, "backup", "daily"),
			want: tagSet(t, "backup", "daily"),
		},
		{
			name: "tags = [] stays an empty set, not null",
			plan: tagSet(t),
			api:  nullTags(),
			want: tagSet(t),
		},
		{
			name:    "tags = [] stays an empty set on refresh",
			plan:    tagSet(t),
			api:     nullTags(),
			refresh: true,
			want:    tagSet(t),
		},
		{
			name: "API matches plan",
			plan: tagSet(t, "environment", "production"),
			api:  tagSet(t, "environment", "production"),
			want: tagSet(t, "environment", "production"),
		},
		{
			name:     "post-apply, API returned no tags",
			plan:     tagSet(t, "environment", "production"),
			api:      nullTags(),
			want:     tagSet(t, "environment", "production"),
			warnings: 1,
		},
		{
			name:     "post-apply, API added a tag",
			plan:     tagSet(t, "environment", "production"),
			api:      tagSet(t, "environment", "production", "backup", "daily"),
			want:     tagSet(t, "environment", "production"),
			warnings: 1,
		},
		{
			name:    "refresh, API returned no tags: drift is reported",
			plan:    tagSet(t, "environment", "production"),
			api:     nullTags(),
			refresh: true,
			want:    nullTags(),
		},
		{
			name:    "refresh, API added a tag: drift is reported",
			plan:    tagSet(t, "environment", "production"),
			api:     tagSet(t, "environment", "production", "backup", "daily"),
			refresh: true,
			want:    tagSet(t, "environment", "production", "backup", "daily"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, diags := reconcileTags(42, tc.plan, tc.api, tc.refresh)
			if diags.HasError() {
				t.Fatalf("unexpected error diagnostics: %v", diags)
			}

			if !got.Equal(tc.want) {
				t.Errorf("tags = %v, want %v", got, tc.want)
			}

			if n := countWarnings(diags); n != tc.warnings {
				t.Errorf("got %d warnings, want %d: %v", n, tc.warnings, diags)
			}
		})
	}
}
