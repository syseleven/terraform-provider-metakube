package resource_sshkey

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func SSHKeySchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The id of the SSH key resource",
			},
			"project_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Reference project identifier",
			},
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Name for the resource",
			},
			"public_key": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					publicKeyNormalizePlanModifier{},
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Public ssh key",
			},
		},
	}
}

// SSHKeyModel represents the Terraform resource model for an SSH key.
type SSHKeyModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	Name      types.String `tfsdk:"name"`
	PublicKey  types.String `tfsdk:"public_key"`
}

type publicKeyNormalizePlanModifier struct{}

func (m publicKeyNormalizePlanModifier) Description(_ context.Context) string {
	return "Suppresses diff for public key whitespace differences"
}

func (m publicKeyNormalizePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m publicKeyNormalizePlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() {
		return
	}
	if strings.TrimSpace(req.StateValue.ValueString()) == strings.TrimSpace(req.PlanValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}
