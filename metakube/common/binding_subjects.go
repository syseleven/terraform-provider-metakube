package common

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/models"
)

type subjectKey struct {
	kind string
	name string
}

type subjectModel struct {
	Kind types.String `tfsdk:"kind"`
	Name types.String `tfsdk:"name"`
}

// RefreshBindingSubjects keeps managed subjects that exist in bindings, in their state order.
// Callers must filter bindings by role and, for namespaced resources, by namespace.
// A null state imports users and groups from the first binding with supported subjects.
func RefreshBindingSubjects(ctx context.Context, state types.List, bindings [][]*models.Subject) (types.List, diag.Diagnostics) {
	available := make(map[subjectKey]struct{})
	var subjects []subjectModel
	for _, binding := range bindings {
		for _, subject := range binding {
			if subject == nil || subject.Name == "" {
				continue
			}
			kind := strings.ToLower(subject.Kind)
			if kind != "user" && kind != "group" {
				continue
			}
			key := subjectKey{kind: kind, name: subject.Name}
			if _, exists := available[key]; exists {
				continue
			}
			available[key] = struct{}{}
			subjects = append(subjects, subjectModel{
				Kind: types.StringValue(kind),
				Name: types.StringValue(subject.Name),
			})
		}
		if state.IsNull() && len(subjects) > 0 {
			break
		}
	}

	if !state.IsNull() {
		var managed []subjectModel
		if diags := state.ElementsAs(ctx, &managed, false); diags.HasError() {
			return state, diags
		}
		subjects = nil
		for _, subject := range managed {
			key := subjectKey{kind: subject.Kind.ValueString(), name: subject.Name.ValueString()}
			if _, exists := available[key]; exists {
				subjects = append(subjects, subject)
			}
		}
	}

	return types.ListValueFrom(ctx, state.ElementType(ctx), subjects)
}
