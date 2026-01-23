package resource_role_binding

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func RoleBindingSchema(ctx context.Context) schema.Schema {
	blocks := metakubeRoleBindingSubjectBlock()
	blocks["timeouts"] = timeouts.Block(ctx, timeouts.Opts{
		Create: true,
	})

	return schema.Schema{
		Attributes: metakubeRoleBindingAttributes(),
		Blocks:     blocks,
	}
}

// RoleBindingModel represents the Terraform resource model for a role binding.
type RoleBindingModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	ClusterID types.String `tfsdk:"cluster_id"`
	Namespace types.String `tfsdk:"namespace"`
	RoleName  types.String `tfsdk:"role_name"`
	Subject   types.List   `tfsdk:"subject"`
	Timeouts  timeouts.Value `tfsdk:"timeouts"`
}

// SubjectModel represents the subject block.
type SubjectModel struct {
	Kind types.String `tfsdk:"kind"`
	Name types.String `tfsdk:"name"`
}

func metakubeSubjectAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"kind": types.StringType,
		"name": types.StringType,
	}
}

func metakubeRoleBindingAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "The id of the role binding resource",
		},
		"project_id": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
			Description: "The id of the project resource belongs to",
		},
		"cluster_id": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
			Description: "The id of the cluster resource belongs to",
		},
		"namespace": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
			Description: "The name of the namespace",
		},
		"role_name": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
			Description: "The name of the role to bind to",
		},
	}
}

func metakubeRoleBindingSubjectBlock() map[string]schema.Block {
	return map[string]schema.Block{
		"subject": schema.ListNestedBlock{
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
			},
			Description: "Users and groups to bind for",
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"kind": schema.StringAttribute{
						Required:    true,
						Description: "Can be either 'user' or 'group'",
						Validators: []validator.String{
							stringvalidator.OneOf("user", "group"),
						},
					},
					"name": schema.StringAttribute{
						Optional:    true,
						Description: "Subject name",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
				},
			},
		},
	}
}
