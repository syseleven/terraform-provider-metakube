package common

import (
	"github.com/go-openapi/runtime"
	"github.com/hashicorp/terraform-plugin-framework/types"
	k8client "github.com/syseleven/go-metakube/client"
	"go.uber.org/zap"
)

type MetaKubeProviderMeta struct {
	Client *k8client.MetaKubeAPI
	Auth   runtime.ClientAuthInfoWriter
	Log    *zap.SugaredLogger
}

type MetakubeProviderConfig struct {
	Host        types.String `tfsdk:"host"`
	Token       types.String `tfsdk:"token"`
	TokenPath   types.String `tfsdk:"token_path"`
	Development types.Bool   `tfsdk:"development"`
	Debug       types.Bool   `tfsdk:"debug"`
	LogPath     types.String `tfsdk:"log_path"`
}
