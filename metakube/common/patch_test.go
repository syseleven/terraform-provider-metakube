package common

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var nestedObjType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"tags": types.MapType{ElemType: types.StringType},
}}

var rootAttrTypes = map[string]attr.Type{
	"labels":          types.MapType{ElemType: types.StringType},
	"server_group_id": types.StringType,
	"nested":          types.ListType{ElemType: nestedObjType},
}

type frameworkRoot struct {
	Labels        types.Map    `tfsdk:"labels"`
	ServerGroupID types.String `tfsdk:"server_group_id"`
	Nested        types.List   `tfsdk:"nested"`
}

type frameworkNested struct {
	Tags types.Map `tfsdk:"tags"`
}

type apiRoot struct {
	Labels        map[string]string `json:"labels,omitempty"`
	ServerGroupID string            `json:"serverGroupID,omitempty"`
	Nested        *apiNested        `json:"nested,omitempty"`
}

type apiNested struct {
	Tags map[string]string `json:"tags,omitempty"`
}

func mapVal(t *testing.T, m map[string]string) types.Map {
	t.Helper()
	if m == nil {
		return types.MapNull(types.StringType)
	}
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(v)
	}
	mv, d := types.MapValue(types.StringType, elems)
	if d.HasError() {
		t.Fatalf("build map: %v", d)
	}
	return mv
}

func nestedList(t *testing.T, tags types.Map) types.List {
	t.Helper()
	obj, d := types.ObjectValue(nestedObjType.AttrTypes, map[string]attr.Value{"tags": tags})
	if d.HasError() {
		t.Fatalf("build nested object: %v", d)
	}
	l, d := types.ListValue(nestedObjType, []attr.Value{obj})
	if d.HasError() {
		t.Fatalf("build nested list: %v", d)
	}
	return l
}

func rootObject(t *testing.T, labels types.Map, serverGroupID types.String, nested types.List) types.Object {
	t.Helper()
	o, d := types.ObjectValue(rootAttrTypes, map[string]attr.Value{
		"labels":          labels,
		"server_group_id": serverGroupID,
		"nested":          nested,
	})
	if d.HasError() {
		t.Fatalf("build root object: %v", d)
	}
	return o
}

func TestAddMergePatchDeletions(t *testing.T) {
	nullNested := types.ListNull(nestedObjType)

	tests := []struct {
		name   string
		patch  map[string]interface{}
		config attr.Value
		plan   attr.Value
		state  attr.Value
		check  func(t *testing.T, patch map[string]interface{})
	}{
		{
			name:   "map add and remove",
			patch:  map[string]interface{}{},
			config: rootObject(t, mapVal(t, map[string]string{"a": "1", "b": "2"}), types.StringNull(), nullNested),
			plan:   rootObject(t, mapVal(t, map[string]string{"a": "1", "b": "2"}), types.StringNull(), nullNested),
			state:  rootObject(t, mapVal(t, map[string]string{"b": "old", "c": "3"}), types.StringNull(), nullNested),
			check: func(t *testing.T, patch map[string]interface{}) {
				got := AsObject(patch["labels"])
				if got["a"] != "1" || got["b"] != "2" {
					t.Fatalf("expected updated keys, got %#v", got)
				}
				if v, ok := got["c"]; !ok || v != nil {
					t.Fatalf("expected removed key c to be null, got %#v", got)
				}
			},
		},
		{
			name:   "equal map is left untouched",
			patch:  map[string]interface{}{"labels": "SET"},
			config: rootObject(t, mapVal(t, map[string]string{"a": "1"}), types.StringNull(), nullNested),
			plan:   rootObject(t, mapVal(t, map[string]string{"a": "1"}), types.StringNull(), nullNested),
			state:  rootObject(t, mapVal(t, map[string]string{"a": "1"}), types.StringNull(), nullNested),
			check: func(t *testing.T, patch map[string]interface{}) {
				if patch["labels"] != "SET" {
					t.Fatalf("expected SET value preserved, got %#v", patch["labels"])
				}
			},
		},
		{
			name:   "omitted optional computed string clears stale state",
			patch:  map[string]interface{}{"serverGroupID": "SET"},
			config: rootObject(t, types.MapNull(types.StringType), types.StringNull(), nullNested),
			plan:   rootObject(t, types.MapNull(types.StringType), types.StringUnknown(), nullNested),
			state:  rootObject(t, types.MapNull(types.StringType), types.StringValue("was-set"), nullNested),
			check: func(t *testing.T, patch map[string]interface{}) {
				if v, ok := patch["serverGroupID"]; !ok || v != nil {
					t.Fatalf("expected serverGroupID nulled, got ok=%v val=%#v", ok, v)
				}
			},
		},
		{
			name:   "scalar still set keeps SET value",
			patch:  map[string]interface{}{"serverGroupID": "new"},
			config: rootObject(t, types.MapNull(types.StringType), types.StringValue("new"), nullNested),
			plan:   rootObject(t, types.MapNull(types.StringType), types.StringValue("new"), nullNested),
			state:  rootObject(t, types.MapNull(types.StringType), types.StringValue("old"), nullNested),
			check: func(t *testing.T, patch map[string]interface{}) {
				if patch["serverGroupID"] != "new" {
					t.Fatalf("expected SET value preserved, got %#v", patch["serverGroupID"])
				}
			},
		},
		{
			name:   "nested object unwrap and map remove",
			patch:  map[string]interface{}{"nested": map[string]interface{}{}},
			config: rootObject(t, types.MapNull(types.StringType), types.StringNull(), nestedList(t, mapVal(t, map[string]string{"a": "1"}))),
			plan:   rootObject(t, types.MapNull(types.StringType), types.StringNull(), nestedList(t, mapVal(t, map[string]string{"a": "1"}))),
			state:  rootObject(t, types.MapNull(types.StringType), types.StringNull(), nestedList(t, mapVal(t, map[string]string{"a": "old", "b": "2"}))),
			check: func(t *testing.T, patch map[string]interface{}) {
				nested := AsObject(patch["nested"])
				tags := AsObject(nested["tags"])
				if tags["a"] != "1" {
					t.Fatalf("expected updated nested tag, got %#v", tags)
				}
				if v, ok := tags["b"]; !ok || v != nil {
					t.Fatalf("expected removed nested tag b to be null, got %#v", tags)
				}
			},
		},
		{
			name:   "list-of-one is unwrapped at top level",
			patch:  map[string]interface{}{},
			config: listOfOne(t, rootObject(t, mapVal(t, map[string]string{"a": "1"}), types.StringNull(), nullNested)),
			plan:   listOfOne(t, rootObject(t, mapVal(t, map[string]string{"a": "1"}), types.StringNull(), nullNested)),
			state:  listOfOne(t, rootObject(t, mapVal(t, map[string]string{"a": "1", "b": "2"}), types.StringNull(), nullNested)),
			check: func(t *testing.T, patch map[string]interface{}) {
				got := AsObject(patch["labels"])
				if v, ok := got["b"]; !ok || v != nil {
					t.Fatalf("expected removed key b to be null, got %#v", got)
				}
			},
		},
		{
			name:   "null state is a no-op",
			patch:  map[string]interface{}{"labels": "SET"},
			config: rootObject(t, mapVal(t, map[string]string{"a": "1"}), types.StringNull(), nullNested),
			plan:   rootObject(t, mapVal(t, map[string]string{"a": "1"}), types.StringNull(), nullNested),
			state:  types.ObjectNull(rootAttrTypes),
			check: func(t *testing.T, patch map[string]interface{}) {
				if patch["labels"] != "SET" {
					t.Fatalf("expected no changes for null state, got %#v", patch)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddMergePatchDeletions(tt.patch, tt.config, tt.plan, tt.state, apiRoot{}, frameworkRoot{}, frameworkNested{})
			tt.check(t, tt.patch)
		})
	}
}

func listOfOne(t *testing.T, obj types.Object) types.List {
	t.Helper()
	l, d := types.ListValue(types.ObjectType{AttrTypes: rootAttrTypes}, []attr.Value{obj})
	if d.HasError() {
		t.Fatalf("build list-of-one: %v", d)
	}
	return l
}

func TestMapMergePatch(t *testing.T) {
	t.Run("nil plan removes all state keys", func(t *testing.T) {
		got := MapMergePatch(types.MapNull(types.StringType), mapVal(t, map[string]string{"a": "1", "b": "2"}))
		if v, ok := got["a"]; !ok || v != nil {
			t.Fatalf("expected a nulled, got %#v", got)
		}
		if v, ok := got["b"]; !ok || v != nil {
			t.Fatalf("expected b nulled, got %#v", got)
		}
	})
	t.Run("nil state keeps plan values", func(t *testing.T) {
		got := MapMergePatch(mapVal(t, map[string]string{"a": "1"}), types.MapNull(types.StringType))
		if got["a"] != "1" {
			t.Fatalf("expected a kept, got %#v", got)
		}
	})
}

func TestAsObject(t *testing.T) {
	if got := AsObject(map[string]interface{}{"a": 1}); got["a"] != 1 {
		t.Fatalf("expected passthrough map, got %#v", got)
	}
	if got := AsObject("not-a-map"); len(got) != 0 {
		t.Fatalf("expected empty map for non-map input, got %#v", got)
	}
	if got := AsObject(nil); got == nil || len(got) != 0 {
		t.Fatalf("expected empty non-nil map for nil input, got %#v", got)
	}
}
