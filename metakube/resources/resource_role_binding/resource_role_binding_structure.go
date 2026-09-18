package resource_role_binding

import (
	"context"

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
