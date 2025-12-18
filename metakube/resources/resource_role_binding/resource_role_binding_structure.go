package resource_role_binding

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/models"
)

func metakubeRoleBindingExpandSubjects(ctx context.Context, list types.List) []models.RoleUser {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var subjectModels []SubjectModel
	if diags := list.ElementsAs(ctx, &subjectModels, false); diags.HasError() || len(subjectModels) == 0 {
		return nil
	}

	var result []models.RoleUser
	for _, subjectModel := range subjectModels {
		if subjectModel.Kind.ValueString() == "user" {
			result = append(result, models.RoleUser{
				UserEmail: subjectModel.Name.ValueString(),
			})
		} else if subjectModel.Kind.ValueString() == "group" {
			result = append(result, models.RoleUser{
				Group: subjectModel.Name.ValueString(),
			})
		}
	}

	return result
}

func metakubeClusterRoleBindingFlattenSubjects(ctx context.Context, roleBindingModel *RoleBindingModel, in []*models.Subject) diag.Diagnostics {
	if len(in) == 0 {
		return nil
	}

	var diags diag.Diagnostics
	var subjectModel SubjectModel

	for _, subject := range in {
		if subject == nil || (subject.Kind == "" || subject.Name == "") {
			roleBindingModel.Subject = types.ListNull(types.ObjectType{AttrTypes: metakubeSubjectAttrTypes()})
			return diags
		}

		subjectModel.Kind = types.StringValue(strings.ToLower(subject.Kind))
		subjectModel.Name = types.StringValue(subject.Name)

		objVal, d := types.ObjectValueFrom(ctx, metakubeSubjectAttrTypes(), subjectModel)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}

		listVal, d := types.ListValue(types.ObjectType{AttrTypes: metakubeSubjectAttrTypes()}, []attr.Value{objVal})
		diags.Append(d...)
		roleBindingModel.Subject = listVal
	}

	return diags
}
