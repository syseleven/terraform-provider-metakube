package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringMapMergePatch builds an RFC 7396 JSON merge patch for a string map.
// Keys in the plan are set to their planned values, while keys present only in
// state are set to null so the API removes them. Null maps are treated as empty;
// unknown maps and null or unknown elements are rejected because they cannot
// produce an unambiguous patch.
func StringMapMergePatch(plan, state types.Map) (map[string]any, error) {
	planElements, err := knownStringMap("plan", plan)
	if err != nil {
		return nil, err
	}
	stateElements, err := knownStringMap("state", state)
	if err != nil {
		return nil, err
	}

	patch := make(map[string]any, len(planElements)+len(stateElements))
	for key, value := range planElements {
		patch[key] = value
	}
	for key := range stateElements {
		if _, exists := planElements[key]; !exists {
			patch[key] = nil
		}
	}
	return patch, nil
}

// knownStringMap converts a known Terraform map into ordinary Go values.
// Keeping this validation at the patch boundary prevents unknown values from
// being mistaken for either updates or deletions.
func knownStringMap(name string, value types.Map) (map[string]string, error) {
	if value.IsUnknown() {
		return nil, fmt.Errorf("%s map is unknown", name)
	}
	if elementType := value.ElementType(context.Background()); elementType == nil || !elementType.Equal(types.StringType) {
		return nil, fmt.Errorf("%s map must contain strings", name)
	}
	if value.IsNull() {
		return map[string]string{}, nil
	}

	elements := make(map[string]string, len(value.Elements()))
	for key, element := range value.Elements() {
		stringElement, ok := element.(types.String)
		if !ok || stringElement.IsNull() || stringElement.IsUnknown() {
			return nil, fmt.Errorf("%s map element %q is not a known string", name, key)
		}
		elements[key] = stringElement.ValueString()
	}
	return elements, nil
}
