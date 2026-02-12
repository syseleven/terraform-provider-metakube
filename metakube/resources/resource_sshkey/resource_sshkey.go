package resource_sshkey

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

var (
	_ resource.Resource                = &metakubeSSHKey{}
	_ resource.ResourceWithConfigure   = &metakubeSSHKey{}
	_ resource.ResourceWithImportState = &metakubeSSHKey{}
)

func NewSSHKey() resource.Resource {
	return &metakubeSSHKey{}
}

type metakubeSSHKey struct {
	meta *common.MetaKubeProviderMeta
}

func (r *metakubeSSHKey) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sshkey"
}

func (r *metakubeSSHKey) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = SSHKeySchema(ctx)
}

func (r *metakubeSSHKey) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	meta, ok := req.ProviderData.(*common.MetaKubeProviderMeta)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.MetaKubeProviderMeta, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.meta = meta
}

func (r *metakubeSSHKey) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SSHKeyModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := project.NewCreateSSHKeyParams()
	p.SetContext(ctx)
	p.SetProjectID(plan.ProjectID.ValueString())
	p.Key = &models.SSHKey{
		Name: plan.Name.ValueString(),
		Spec: &models.SSHKeySpec{
			PublicKey: plan.PublicKey.ValueString(),
		},
	}

	created, err := r.meta.Client.Project.CreateSSHKey(p, r.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to create SSH key: %s", common.StringifyResponseError(err)))
		return
	}

	plan.ID = types.StringValue(created.Payload.ID)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *metakubeSSHKey) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SSHKeyModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sshkey, projectID, err := r.findSSHKeyByID(ctx, data.ID.ValueString(), data.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Error while reading SSH key: %s", err))
		return
	}
	if sshkey == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ProjectID = types.StringValue(projectID)
	data.Name = types.StringValue(sshkey.Name)
	data.PublicKey = types.StringValue(sshkey.Spec.PublicKey)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *metakubeSSHKey) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes require replacement, so Update is never called.
}

func (r *metakubeSSHKey) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SSHKeyModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := project.NewDeleteSSHKeyParams()
	p.SetContext(ctx)
	p.SetProjectID(state.ProjectID.ValueString())
	p.SetSSHKeyID(state.ID.ValueString())

	_, err := r.meta.Client.Project.DeleteSSHKey(p, r.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to delete SSH key: %s", common.StringifyResponseError(err)))
		return
	}
}

func (r *metakubeSSHKey) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	switch len(parts) {
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	case 1:
		projectID, err := r.findProjectForSSHKey(ctx, parts[0])
		if err != nil {
			resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Error finding project for SSH key: %s", err))
			return
		}
		if projectID == "" {
			resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Could not find project owning SSH key '%s'", parts[0]))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	default:
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Please provide resource identifier in format 'project_id:sshkey_id' or 'sshkey_id'",
		)
	}
}

func (r *metakubeSSHKey) findSSHKeyByID(ctx context.Context, id, projectID string) (*models.SSHKey, string, error) {
	if projectID == "" {
		var err error
		projectID, err = r.findProjectForSSHKey(ctx, id)
		if err != nil {
			return nil, "", err
		}
		if projectID == "" {
			return nil, "", nil
		}
	}

	const readTimeout = 5 * time.Minute

	var foundKey *models.SSHKey
	err := common.RetryContext(ctx, readTimeout, func() *common.RetryError {
		prms := project.NewListSSHKeysParams().WithContext(ctx).WithProjectID(projectID)
		res, err := r.meta.Client.Project.ListSSHKeys(prms, r.meta.Auth)
		if err != nil {
			if _, ok := err.(*project.ListSSHKeysForbidden); ok {
				return common.RetryableError(fmt.Errorf("waiting for RBAC permissions: %s", common.StringifyResponseError(err)))
			}
			return common.NonRetryableError(fmt.Errorf("list ssh keys: %s", common.StringifyResponseError(err)))
		}

		for _, k := range res.Payload {
			if k.ID == id {
				foundKey = k
				return nil
			}
		}

		return nil
	})
	if err != nil {
		r.meta.Log.Debugf("error while waiting for the SSH keys: %v", err)
		return nil, "", fmt.Errorf("error while waiting for the SSH keys: %v", err)
	}

	return foundKey, projectID, nil
}

func (r *metakubeSSHKey) findProjectForSSHKey(ctx context.Context, id string) (string, error) {
	res, err := r.meta.Client.Project.ListProjects(project.NewListProjectsParams(), r.meta.Auth)
	if err != nil {
		return "", fmt.Errorf("list projects: %v", err)
	}

	for _, prj := range res.Payload {
		ok, err := r.sshKeyBelongsToProject(ctx, prj.ID, id)
		if ok {
			return prj.ID, nil
		}
		if err != nil {
			return "", err
		}
	}

	r.meta.Log.Infof("owner project for SSH key with id(%s) not found", id)
	return "", nil
}

func (r *metakubeSSHKey) sshKeyBelongsToProject(ctx context.Context, projectID, id string) (bool, error) {
	prms := project.NewListSSHKeysParams().WithContext(ctx).WithProjectID(projectID)
	res, err := r.meta.Client.Project.ListSSHKeys(prms, r.meta.Auth)
	if err != nil {
		return false, fmt.Errorf("list sshkeys: %s", common.StringifyResponseError(err))
	}

	for _, k := range res.Payload {
		if k.ID == id {
			return true, nil
		}
	}

	return false, nil
}
