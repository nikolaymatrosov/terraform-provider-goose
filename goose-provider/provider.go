package goose_provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"terraform-provider-goose/common"
	goose_ydb_migration "terraform-provider-goose/goose-provider/goose-ydb-migration"
	"terraform-provider-goose/goose-provider/provider-config"

	"github.com/hashicorp/terraform-plugin-framework-validators/providervalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Provider struct {
	emptyFolder bool
	config      provider_config.Config
}

func (p Provider) Metadata(_ context.Context, _ provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "goose"
}

func (p Provider) Schema(_ context.Context, _ provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: common.Descriptions["endpoint"],
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: common.Descriptions["token"],
			},
			"service_account_key_file": schema.StringAttribute{
				Optional:    true,
				Description: common.Descriptions["service_account_key_file"],
				Validators: []validator.String{
					saKeyValidator{},
				},
			},
			"max_retries": schema.Int64Attribute{
				Optional:    true,
				Description: common.Descriptions["max_retries"],
			},
		},
	}
}

func (p Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	p.config = provider_config.Config{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &p.config.ProviderState)...)
	p.config.UserAgent = types.StringValue(req.TerraformVersion)
	p.config.ProviderState = setDefaults(p.config.ProviderState)

	if err := p.config.InitAndValidate(ctx, req.TerraformVersion, false); err != nil {
		resp.Diagnostics.AddError("Failed to configure", err.Error())
	}
	resp.ResourceData = &p.config
	resp.DataSourceData = &p.config
}

func (p Provider) ConfigValidators(_ context.Context) []provider.ConfigValidator {
	return []provider.ConfigValidator{
		providervalidator.Conflicting(
			path.MatchRoot("token"),
			path.MatchRoot("service_account_key_file"),
		),
	}
}

func (p Provider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p Provider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		goose_ydb_migration.NewResource,
	}
}

func NewFrameworkProvider() provider.Provider {
	return &Provider{}
}

type saKeyValidator struct{}

func (v saKeyValidator) Description(_ context.Context) string {
	return fmt.Sprintf("Validate Service Account Key")
}

func (v saKeyValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("Validate Service Account Key")
}

func (v saKeyValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	saKey := req.ConfigValue.ValueString()
	if len(saKey) == 0 {
		return
	}
	if _, err := os.Stat(saKey); err == nil {
		return
	}
	var _f map[string]interface{}
	if err := json.Unmarshal([]byte(saKey), &_f); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid SA Key",
			fmt.Sprintf("JSON in %q are not valid: %s", saKey, err),
		)
	}
}

func setToDefaultIfNeeded(field types.String, osEnvName string, defaultVal string) types.String {
	if len(field.ValueString()) != 0 {
		return field
	}
	field = types.StringValue(os.Getenv(osEnvName))
	if len(field.ValueString()) == 0 {
		field = types.StringValue(defaultVal)
	}
	return field
}

func setDefaults(config provider_config.State) provider_config.State {
	config.Endpoint = setToDefaultIfNeeded(config.Endpoint, "YC_ENDPOINT", common.DefaultEndpoint)
	config.Token = setToDefaultIfNeeded(config.Token, "YC_TOKEN", "")
	config.ServiceAccountKeyFileOrContent = setToDefaultIfNeeded(config.ServiceAccountKeyFileOrContent, "YC_SERVICE_ACCOUNT_KEY_FILE", "")

	if config.MaxRetries.IsUnknown() || config.MaxRetries.IsNull() {
		config.MaxRetries = types.Int64Value(common.DefaultMaxRetries)
	}

	return config
}
