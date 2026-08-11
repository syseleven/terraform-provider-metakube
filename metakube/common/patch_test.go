package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringMapMergePatch(t *testing.T) {
	plan := types.MapValueMust(types.StringType, map[string]attr.Value{
		"added":   types.StringValue("new"),
		"changed": types.StringValue("new"),
	})
	state := types.MapValueMust(types.StringType, map[string]attr.Value{
		"changed": types.StringValue("old"),
		"removed": types.StringValue("old"),
	})

	got, err := StringMapMergePatch(plan, state)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"added": "new", "changed": "new", "removed": nil}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("patch mismatch (-want +got):\n%s", diff)
	}
}

func TestStringMapMergePatchRejectsUnknown(t *testing.T) {
	known := types.MapValueMust(types.StringType, map[string]attr.Value{})
	if _, err := StringMapMergePatch(types.MapUnknown(types.StringType), known); err == nil {
		t.Fatal("expected unknown map error")
	}
	if _, err := StringMapMergePatch(
		types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringUnknown()}),
		known,
	); err == nil {
		t.Fatal("expected unknown element error")
	}
}
