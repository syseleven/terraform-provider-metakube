package common

import (
	"reflect"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AddMergePatchDeletions records field deletions in an already-built SET patch.
//
// When a user removes an optional field from their Terraform config, the field
// should be deleted in the API. A SET patch can only set values, so a removed
// field simply disappears from the patch and the API never learns it should go
// away. JSON merge patch fixes this. Setting a field to null tells the API to
// delete it. This function finds those removals and adds the nulls.
//
// To decide what was removed, it compares the field across config, plan, and
// state for the object described by apiModel, then translates the Terraform
// attribute name into the API model's json tag. If a single object is wrapped in
// a list or set, it is unwrapped first.
//
// Only deletions are added; the rest of the SET patch is left untouched. A field
// is skipped whenever it cannot be compared safely, for example when a parent is
// null or unknown, the types do not line up, or the API model has no matching
// json tag.
func AddMergePatchDeletions(patch map[string]any, config, plan, state attr.Value, apiModel any, frameworkModels ...any) {
	if patch == nil {
		return
	}
	fwTypes := frameworkModelTypes(frameworkModels)
	addMergePatchDeletions(patch, config, plan, state, bestFrameworkType(indirectType(reflect.TypeOf(apiModel)), fwTypes), indirectType(reflect.TypeOf(apiModel)), fwTypes)
}

func addMergePatchDeletions(patch map[string]any, config, plan, state attr.Value, fwType, apiType reflect.Type, fwTypes []reflect.Type) {
	configAttrs, ok := objectAttributes(config)
	if !ok {
		return
	}
	planAttrs, ok := objectAttributes(plan)
	if !ok {
		return
	}
	stateAttrs, ok := objectAttributes(state)
	if !ok {
		return
	}
	fields := mergePatchFields(fwType, apiType)
	if len(fields) == 0 {
		return
	}

	for attrName, planVal := range planAttrs {
		field, ok := fields[attrName]
		if !ok {
			continue
		}
		configVal := configAttrs[attrName]
		stateVal := stateAttrs[attrName]
		jsonKey := field.JSONKey
		if jsonKey == "" {
			continue
		}

		if planMap, ok := planVal.(types.Map); ok {
			stateMap, ok := stateVal.(types.Map)
			if ok && !planMap.Equal(stateMap) {
				patch[jsonKey] = MapMergePatch(planMap, stateMap)
			}
			continue
		}

		if shouldClearOmittedOptionalComputedString(configVal, planVal, stateVal) {
			patch[jsonKey] = nil
			continue
		}

		childAPIType, ok := nestedObjectAPIType(field.APIField.Type)
		if !ok {
			continue
		}
		child := AsObject(patch[jsonKey])
		addMergePatchDeletions(child, configVal, planVal, stateVal, bestFrameworkType(childAPIType, fwTypes), childAPIType, fwTypes)
		if len(child) > 0 {
			patch[jsonKey] = child
		}
	}
}

// MapMergePatch builds a JSON merge-patch fragment from a plan and state map. Keys present
// in plan are set to their plan value, and keys present in state but absent from plan are
// set to null so the API removes them.
func MapMergePatch(planMap, stateMap types.Map) map[string]any {
	result := make(map[string]any)

	planElements := map[string]attr.Value{}
	if !planMap.IsNull() && !planMap.IsUnknown() {
		planElements = planMap.Elements()
		for k, v := range planElements {
			if s, ok := v.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
				result[k] = s.ValueString()
			}
		}
	}

	if !stateMap.IsNull() && !stateMap.IsUnknown() {
		for k := range stateMap.Elements() {
			if _, exists := planElements[k]; !exists {
				result[k] = nil
			}
		}
	}

	return result
}

func AsObject(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func objectAttributes(v attr.Value) (map[string]attr.Value, bool) {
	switch t := v.(type) {
	case types.Object:
		if t.IsNull() || t.IsUnknown() {
			return nil, false
		}
		return t.Attributes(), true
	case types.List:
		return singleObjectAttributes(t.IsNull(), t.IsUnknown(), t.Elements())
	case types.Set:
		return singleObjectAttributes(t.IsNull(), t.IsUnknown(), t.Elements())
	default:
		return nil, false
	}
}

func singleObjectAttributes(isNull, isUnknown bool, elems []attr.Value) (map[string]attr.Value, bool) {
	if isNull || isUnknown || len(elems) != 1 {
		return nil, false
	}
	obj, ok := elems[0].(types.Object)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return nil, false
	}
	return obj.Attributes(), true
}


// True when an optional computed string was removed from config but still has a value in state.
func shouldClearOmittedOptionalComputedString(configVal, planVal, stateVal attr.Value) bool {
	if configVal == nil || planVal == nil || stateVal == nil {
		return false
	}
	if !configVal.IsNull() || !planVal.IsUnknown() || stateVal.IsNull() || stateVal.IsUnknown() {
		return false
	}
	stateString, ok := stateVal.(types.String)
	return ok && stateString.ValueString() != ""
}

var (
	mergePatchFieldsMu     sync.RWMutex
	mergePatchFieldsByType = map[struct {
		Framework reflect.Type
		API       reflect.Type
	}]map[string]mergePatchField{}
)

type mergePatchField struct {
	JSONKey  string
	APIField reflect.StructField
}

func indirectType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func nestedObjectAPIType(t reflect.Type) (reflect.Type, bool) {
	t = indirectType(t)
	if t == nil || t.Kind() != reflect.Struct {
		return nil, false
	}
	return t, true
}

func frameworkModelTypes(models []any) []reflect.Type {
	result := make([]reflect.Type, 0, len(models))
	for _, model := range models {
		t := indirectType(reflect.TypeOf(model))
		if t != nil && t.Kind() == reflect.Struct {
			result = append(result, t)
		}
	}
	return result
}


// HACK: pick the framework model whose fields best match apiType when multiple types are passed.
func bestFrameworkType(apiType reflect.Type, fwTypes []reflect.Type) reflect.Type {
	var best reflect.Type
	bestScore := 0
	for _, fwType := range fwTypes {
		score := modelMatchScore(fwType, apiType)
		if score > bestScore {
			best = fwType
			bestScore = score
		}
	}
	return best
}

func modelMatchScore(fwType, apiType reflect.Type) int {
	if fwType == nil || apiType == nil {
		return 0
	}
	apiByName := map[string]struct{}{}
	apiByTag := map[string]struct{}{}
	for i := 0; i < apiType.NumField(); i++ {
		field := apiType.Field(i)
		apiByName[field.Name] = struct{}{}
		if key := jsonKeyForStructField(field); key != "" {
			apiByTag[canonicalTagName(key)] = struct{}{}
		}
	}

	score := 0
	for i := 0; i < fwType.NumField(); i++ {
		field := fwType.Field(i)
		if _, ok := apiByName[field.Name]; ok {
			score += 2
			continue
		}
		if tag := tfsdkTagForStructField(field); tag != "" {
			if _, ok := apiByTag[canonicalTagName(tag)]; ok {
				score++
			}
		}
	}
	return score
}

func mergePatchFields(fwType, apiType reflect.Type) map[string]mergePatchField {
	if fwType == nil || apiType == nil {
		return nil
	}

	key := struct {
		Framework reflect.Type
		API       reflect.Type
	}{Framework: fwType, API: apiType}

	mergePatchFieldsMu.RLock()
	fields, ok := mergePatchFieldsByType[key]
	mergePatchFieldsMu.RUnlock()
	if ok {
		return fields
	}

	apiByName := map[string]reflect.StructField{}
	apiByTag := map[string]reflect.StructField{}
	for i := 0; i < apiType.NumField(); i++ {
		field := apiType.Field(i)
		jsonKey := jsonKeyForStructField(field)
		if jsonKey == "" {
			continue
		}
		apiByName[field.Name] = field
		apiByTag[canonicalTagName(jsonKey)] = field
	}

	fields = make(map[string]mergePatchField)
	for i := 0; i < fwType.NumField(); i++ {
		fwField := fwType.Field(i)
		tfsdkTag := tfsdkTagForStructField(fwField)
		if tfsdkTag == "" {
			continue
		}

		apiField, ok := apiByName[fwField.Name]
		if !ok {
			apiField, ok = apiByTag[canonicalTagName(tfsdkTag)]
			if !ok {
				continue
			}
		}

		jsonKey := jsonKeyForStructField(apiField)
		if jsonKey == "" {
			continue
		}
		fields[tfsdkTag] = mergePatchField{
			JSONKey:  jsonKey,
			APIField: apiField,
		}
	}

	mergePatchFieldsMu.Lock()
	mergePatchFieldsByType[key] = fields
	mergePatchFieldsMu.Unlock()

	return fields
}

func jsonKeyForStructField(field reflect.StructField) string {
	return tagName(field.Tag.Get("json"))
}

func tfsdkTagForStructField(field reflect.StructField) string {
	return tagName(field.Tag.Get("tfsdk"))
}

func tagName(tag string) string {
	if tag == "" {
		return ""
	}
	name := strings.Split(tag, ",")[0]
	if name == "-" {
		return ""
	}
	return name
}

func canonicalTagName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '_' || r == '-' || r == '.':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}
