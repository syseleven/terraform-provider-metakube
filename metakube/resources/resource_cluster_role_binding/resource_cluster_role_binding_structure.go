package resource_cluster_role_binding

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/models"
)

func metakubeClusterRoleBindingExpandSubjects(ctx context.Context, list types.List) []models.ClusterRoleUser {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var subjectModels []SubjectModel
	if diags := list.ElementsAs(ctx, &subjectModels, false); diags.HasError() || len(subjectModels) == 0 {
		return nil
	}

	var result []models.ClusterRoleUser
	for _, subjectModel := range subjectModels {
		if subjectModel.Kind.ValueString() == "user" {
			result = append(result, models.ClusterRoleUser{
				UserEmail: subjectModel.Name.ValueString(),
			})
		} else if subjectModel.Kind.ValueString() == "group" {
			result = append(result, models.ClusterRoleUser{
				Group: subjectModel.Name.ValueString(),
			})
		}
	}

	return result
}
