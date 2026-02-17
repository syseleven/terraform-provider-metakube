package resource_cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/client/openstack"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/client/versions"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

type metakubeResourceClusterOpenstackValidationData struct {
	dcName                       *string
	domain                       *string
	username                     *string
	password                     *string
	projectID                    *string
	projectName                  *string
	applicationCredentialsID     *string
	applicationCredentialsSecret *string
	network                      *string
	subnetID                     *string
}

type metakubeResourceClusterGeneralOpenstackRequestParams interface {
	SetDatacenterName(*string)
	SetDomain(*string)
	SetUsername(*string)
	SetPassword(*string)
	SetTenantID(*string)
	SetTenant(*string)
	SetApplicationCredentialID(*string)
	SetApplicationCredentialSecret(*string)
	SetContext(context.Context)
}

func (data *metakubeResourceClusterOpenstackValidationData) setParams(ctx context.Context, p metakubeResourceClusterGeneralOpenstackRequestParams) {
	p.SetDatacenterName(data.dcName)
	p.SetDomain(data.domain)
	p.SetUsername(data.username)
	p.SetPassword(data.password)
	p.SetTenantID(data.projectID)
	p.SetTenant(data.projectName)
	p.SetApplicationCredentialID(data.applicationCredentialsID)
	p.SetApplicationCredentialSecret(data.applicationCredentialsSecret)
	p.SetContext(ctx)
}

func newOpenstackValidationData(ctx context.Context, model *ClusterModel) metakubeResourceClusterOpenstackValidationData {
	data := metakubeResourceClusterOpenstackValidationData{
		domain: common.StrToPtr("Default"),
	}

	if model == nil {
		return data
	}

	data.dcName = stringValueToPtr(model.DCName)

	var specs []ClusterSpecModel
	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return data
	}
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return data
	}

	var clouds []ClusterCloudSpecModel
	if specs[0].Cloud.IsNull() || specs[0].Cloud.IsUnknown() {
		return data
	}
	if diags := specs[0].Cloud.ElementsAs(ctx, &clouds, false); diags.HasError() || len(clouds) == 0 {
		return data
	}

	var openstacks []OpenstackCloudSpecModel
	if clouds[0].Openstack.IsNull() || clouds[0].Openstack.IsUnknown() {
		return data
	}
	if diags := clouds[0].Openstack.ElementsAs(ctx, &openstacks, false); diags.HasError() || len(openstacks) == 0 {
		return data
	}

	os := openstacks[0]
	data.network = stringValueToPtr(os.Network)
	data.subnetID = stringValueToPtr(os.SubnetID)

	var userCreds []OpenstackUserCredentialsModel
	if !os.UserCredentials.IsNull() && !os.UserCredentials.IsUnknown() {
		if diags := os.UserCredentials.ElementsAs(ctx, &userCreds, false); !diags.HasError() && len(userCreds) > 0 {
			uc := userCreds[0]
			data.username = stringValueToPtr(uc.Username)
			data.password = stringValueToPtr(uc.Password)
			data.projectID = stringValueToPtr(uc.ProjectID)
			data.projectName = stringValueToPtr(uc.ProjectName)
		}
	}

	var appCreds []OpenstackApplicationCredentialsModel
	if !os.ApplicationCredentials.IsNull() && !os.ApplicationCredentials.IsUnknown() {
		if diags := os.ApplicationCredentials.ElementsAs(ctx, &appCreds, false); !diags.HasError() && len(appCreds) > 0 {
			ac := appCreds[0]
			data.applicationCredentialsID = stringValueToPtr(ac.ID)
			data.applicationCredentialsSecret = stringValueToPtr(ac.Secret)
		}
	}

	return data
}

func stringValueToPtr(s types.String) *string {
	if s.IsUnknown() || s.IsNull() {
		return nil
	}
	val := s.ValueString()
	if val == "" {
		return nil
	}
	return &val
}

func metakubeResourceClusterValidateClusterFields(ctx context.Context, model *ClusterModel, k *common.MetaKubeProviderMeta, isUpdate bool) diag.Diagnostics {
	var ret diag.Diagnostics

	ret.Append(metakubeResourceValidateVersionExistence(ctx, model, k, isUpdate)...)

	if !hasOpenstackConfig(ctx, model) {
		return ret
	}

	data := newOpenstackValidationData(ctx, model)
	hasAuthData := (data.username == nil || *data.username != "") && (data.applicationCredentialsID == nil || *data.applicationCredentialsSecret != "")
	if hasAuthData {
		ret.Append(metakubeResourceClusterValidateFloatingIPPool(ctx, model, k)...)
		ret.Append(metakubeResourceClusterValidateOpenstackNetwork(ctx, model, k)...)
		ret.Append(diagnoseOpenstackSubnetWithIDExistsIfSet(ctx, model, k)...)
	}
	ret.Append(metakubeResourceClusterValidateAccessCredentialsSet(ctx, model)...)
	return ret
}

func hasOpenstackConfig(ctx context.Context, model *ClusterModel) bool {
	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return false
	}
	var specs []ClusterSpecModel
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return false
	}
	if specs[0].Cloud.IsNull() || specs[0].Cloud.IsUnknown() {
		return false
	}
	var clouds []ClusterCloudSpecModel
	if diags := specs[0].Cloud.ElementsAs(ctx, &clouds, false); diags.HasError() || len(clouds) == 0 {
		return false
	}
	if clouds[0].Openstack.IsNull() || clouds[0].Openstack.IsUnknown() {
		return false
	}
	var openstacks []OpenstackCloudSpecModel
	if diags := clouds[0].Openstack.ElementsAs(ctx, &openstacks, false); diags.HasError() || len(openstacks) == 0 {
		return false
	}
	return true
}

func metakubeResourceClusterValidateVersionUpgrade(ctx context.Context, projectID, newVersion string, cluster *models.Cluster, k *common.MetaKubeProviderMeta) diag.Diagnostics {
	p := project.NewGetClusterUpgradesV2Params().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(cluster.ID)
	r, err := k.Client.Project.GetClusterUpgradesV2(p, k.Auth)
	if err != nil {
		var ret diag.Diagnostics
		ret.AddError("Failed to get cluster upgrades", err.Error())
		return ret
	}
	var available []string
	for _, item := range r.Payload {
		available = append(available, item.Version)
		if item.Version == newVersion {
			return nil
		}
	}
	var ret diag.Diagnostics
	ret.AddAttributeError(
		path.Root("spec").AtListIndex(0).AtName("version"),
		fmt.Sprintf("Not allowed upgrade %s->%s", cluster.Spec.Version, newVersion),
		fmt.Sprintf("Please select one of available upgrades: %v", available),
	)
	return ret
}

func metakubeResourceValidateVersionExistence(ctx context.Context, model *ClusterModel, k *common.MetaKubeProviderMeta, isUpdate bool) diag.Diagnostics {
	if isUpdate {
		return nil
	}

	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return nil
	}

	var specs []ClusterSpecModel
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return nil
	}

	version := specs[0].Version.ValueString()
	if version == "" {
		return nil
	}

	p := versions.NewGetMasterVersionsParams().WithContext(ctx)
	r, err := k.Client.Versions.GetMasterVersions(p, k.Auth)
	if err != nil {
		var ret diag.Diagnostics
		ret.AddError("Failed to get available versions", common.StringifyResponseError(err))
		return ret
	}

	available := make([]string, 0)
	for _, v := range r.Payload {
		available = append(available, v.Version)
		if v.Version == version {
			return nil
		}
	}

	var ret diag.Diagnostics
	ret.AddAttributeError(
		path.Root("spec").AtListIndex(0).AtName("version"),
		fmt.Sprintf("Unknown version %s", version),
		fmt.Sprintf("Please select one of available versions: %v", available),
	)
	return ret
}

func metakubeResourceClusterValidateFloatingIPPool(ctx context.Context, model *ClusterModel, k *common.MetaKubeProviderMeta) diag.Diagnostics {
	data := newOpenstackValidationData(ctx, model)

	floatingIPPool, ok := getOpenstackFieldString(ctx, model, func(os OpenstackCloudSpecModel) types.String {
		return os.FloatingIPPool
	})
	if !ok || floatingIPPool == "" {
		return nil
	}

	_, allNets, err := getNetwork(ctx, k, data, floatingIPPool, true)
	if err != nil {
		var diagnoseDetail string
		if len(allNets) > 0 {
			names := make([]string, 0)
			for _, n := range allNets {
				if n.External {
					names = append(names, n.Name)
				}
			}
			diagnoseDetail = fmt.Sprintf("We found following floating IP pools: %v", names)
		}
		var ret diag.Diagnostics
		ret.AddAttributeError(
			path.Root("spec").AtListIndex(0).AtName("cloud").AtListIndex(0).AtName("openstack").AtListIndex(0).AtName("floating_ip_pool"),
			fmt.Sprintf("Invalid value: %v", err),
			diagnoseDetail,
		)
		return ret
	}
	return nil
}

func metakubeResourceClusterValidateOpenstackNetwork(ctx context.Context, model *ClusterModel, k *common.MetaKubeProviderMeta) diag.Diagnostics {
	data := newOpenstackValidationData(ctx, model)

	network, ok := getOpenstackFieldString(ctx, model, func(os OpenstackCloudSpecModel) types.String {
		return os.Network
	})
	if !ok || network == "" {
		return nil
	}

	_, allNets, err := getNetwork(ctx, k, data, network, false)
	if err != nil {
		names := make([]string, 0)
		for _, n := range allNets {
			if !n.External {
				names = append(names, n.Name)
			}
		}
		var diagnoseDetail string
		if len(names) > 0 {
			diagnoseDetail = fmt.Sprintf("We found following networks: %v", names)
		}
		var ret diag.Diagnostics
		ret.AddAttributeError(
			path.Root("spec").AtListIndex(0).AtName("cloud").AtListIndex(0).AtName("openstack").AtListIndex(0).AtName("network"),
			fmt.Sprintf("Invalid value: %v", err),
			diagnoseDetail,
		)
		return ret
	}
	return nil
}

func diagnoseOpenstackSubnetWithIDExistsIfSet(ctx context.Context, model *ClusterModel, k *common.MetaKubeProviderMeta) diag.Diagnostics {
	data := newOpenstackValidationData(ctx, model)
	if data.network == nil || data.subnetID == nil {
		return nil
	}
	network, _, err := getNetwork(ctx, k, data, *data.network, true)
	if err != nil {
		return nil
	}

	subnets, ok, err := getSubnet(ctx, k, data, network.ID)
	if ok {
		return nil
	}
	var diagnoseDetail string
	if len(subnets) > 0 {
		tmp := make([]string, 0)
		for _, i := range subnets {
			tmp = append(tmp, fmt.Sprintf("%s/%s", i.Name, i.ID))
		}
		diagnoseDetail = fmt.Sprintf("We found following subnets (name/id): %v", tmp)
	}
	var ret diag.Diagnostics
	ret.AddAttributeError(
		path.Root("spec").AtListIndex(0).AtName("cloud").AtListIndex(0).AtName("openstack").AtListIndex(0).AtName("subnet_id"),
		fmt.Sprintf("Invalid value: %v", err),
		diagnoseDetail,
	)
	return ret
}

func getOpenstackFieldString(ctx context.Context, model *ClusterModel, getter func(OpenstackCloudSpecModel) types.String) (string, bool) {
	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return "", false
	}
	var specs []ClusterSpecModel
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return "", false
	}
	if specs[0].Cloud.IsNull() || specs[0].Cloud.IsUnknown() {
		return "", false
	}
	var clouds []ClusterCloudSpecModel
	if diags := specs[0].Cloud.ElementsAs(ctx, &clouds, false); diags.HasError() || len(clouds) == 0 {
		return "", false
	}
	if clouds[0].Openstack.IsNull() || clouds[0].Openstack.IsUnknown() {
		return "", false
	}
	var openstacks []OpenstackCloudSpecModel
	if diags := clouds[0].Openstack.ElementsAs(ctx, &openstacks, false); diags.HasError() || len(openstacks) == 0 {
		return "", false
	}
	field := getter(openstacks[0])
	if field.IsNull() || field.IsUnknown() {
		return "", false
	}
	return field.ValueString(), true
}

func getNetwork(ctx context.Context, k *common.MetaKubeProviderMeta, data metakubeResourceClusterOpenstackValidationData, name string, external bool) (*models.OpenstackNetwork, []*models.OpenstackNetwork, error) {
	p := openstack.NewListOpenstackNetworksParams()
	data.setParams(ctx, p)
	res, err := k.Client.Openstack.ListOpenstackNetworks(p, k.Auth)
	if err != nil {
		return nil, nil, fmt.Errorf("find network instance %v", common.StringifyResponseError(err))
	}
	ret := findNetwork(res.Payload, name, external)
	if ret == nil {
		return nil, res.Payload, fmt.Errorf("network `%s` not found", name)
	}
	return ret, res.Payload, nil
}

func findNetwork(list []*models.OpenstackNetwork, network string, external bool) *models.OpenstackNetwork {
	for _, item := range list {
		if item.Name == network && item.External == external {
			return item
		}
	}
	return nil
}

func getSubnet(ctx context.Context, k *common.MetaKubeProviderMeta, data metakubeResourceClusterOpenstackValidationData, networkID string) ([]*models.OpenstackSubnet, bool, error) {
	p := openstack.NewListOpenstackSubnetsParams()
	data.setParams(ctx, p)
	p.SetNetworkID(&networkID)
	res, err := k.Client.Openstack.ListOpenstackSubnets(p, k.Auth)
	if err != nil {
		return nil, false, fmt.Errorf("list network subnets: %v", common.StringifyResponseError(err))
	}
	return res.Payload, findSubnet(res.Payload, *data.subnetID) != nil, nil
}

func findSubnet(list []*models.OpenstackSubnet, id string) *models.OpenstackSubnet {
	for _, item := range list {
		if item.ID == id {
			return item
		}
	}
	return nil
}

func metakubeResourceClusterValidateAccessCredentialsSet(ctx context.Context, model *ClusterModel) diag.Diagnostics {
	data := newOpenstackValidationData(ctx, model)

	username := data.username != nil && *data.username != ""
	password := data.password != nil && *data.password != ""
	projectID := data.projectID != nil && *data.projectID != ""
	projectName := data.projectName != nil && *data.projectName != ""

	applicationCredentialsID := data.applicationCredentialsID != nil && *data.applicationCredentialsID != ""
	applicationCredentialsSecret := data.applicationCredentialsSecret != nil && *data.applicationCredentialsSecret != ""

	usingUsername := username || password || projectID || projectName
	usingApplicationCredentials := applicationCredentialsID || applicationCredentialsSecret

	if usingUsername && (!username || !password || !projectID) {
		var details []string
		if !username {
			details = append(details, "username not set")
		}
		if !password {
			details = append(details, "password not set")
		}
		if !projectID {
			details = append(details, "project_id not set")
		}
		var ret diag.Diagnostics
		ret.AddAttributeError(
			path.Root("spec").AtListIndex(0).AtName("cloud").AtListIndex(0).AtName("openstack").AtListIndex(0),
			"Please set all username, password, project_id and project_name or use application_credentials",
			strings.Join(details, ", "),
		)
		return ret
	}

	if usingApplicationCredentials && (!applicationCredentialsID || !applicationCredentialsSecret) {
		var details []string
		if !applicationCredentialsID {
			details = append(details, "id not set")
		}
		if !applicationCredentialsSecret {
			details = append(details, "secret not set")
		}
		var ret diag.Diagnostics
		ret.AddAttributeError(
			path.Root("spec").AtListIndex(0).AtName("cloud").AtListIndex(0).AtName("openstack").AtListIndex(0),
			"Please set both id and secret",
			strings.Join(details, ", "),
		)
		return ret
	}

	return nil
}
