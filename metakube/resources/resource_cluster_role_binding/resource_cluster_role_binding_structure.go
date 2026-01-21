package resource_cluster_role_binding

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

func metakubeClusterRoleBindingFlattenSubjects(ctx context.Context, clusterRoleBindingModel *ClusterRoleBindingModel, in []*models.Subject) diag.Diagnostics {
	if len(in) == 0 {
		return nil
	}

	var diags diag.Diagnostics
	subjectVals := make([]attr.Value, 0, len(in))

	for _, subject := range in {
		if subject == nil || (subject.Kind == "" || subject.Name == "") {
			clusterRoleBindingModel.Subject = types.ListNull(types.ObjectType{AttrTypes: metakubeSubjectAttrTypes()})
			return diags
		}

		subjectModel := SubjectModel{
			Kind: types.StringValue(strings.ToLower(subject.Kind)),
			Name: types.StringValue(subject.Name),
		}
		objVal, d := types.ObjectValueFrom(ctx, metakubeSubjectAttrTypes(), subjectModel)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}

		subjectVals = append(subjectVals, objVal)
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: metakubeSubjectAttrTypes()}, subjectVals)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	clusterRoleBindingModel.Subject = listVal

	return diags
}
