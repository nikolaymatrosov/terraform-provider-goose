package goose_ydb_migration

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"terraform-provider-goose/common"
	provider_config "terraform-provider-goose/goose-provider/provider-config"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pressly/goose/v3"
	"github.com/yandex-cloud/terraform-provider-yandex/yandex-framework/utils"
	_ "github.com/ydb-platform/ydb-go-sdk/v3"
)

type ydbMigration struct {
	providerConfig *provider_config.Config
}

func NewResource() resource.Resource {
	return &ydbMigration{}
}

func (y *ydbMigration) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "goose_ydb_migration"
}

func (y *ydbMigration) Schema(ctx context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required: true,
			},
			"database": schema.StringAttribute{
				Required: true,
			},
			"tls_enabled": schema.BoolAttribute{
				Optional: true,
			},
			"migration_table": schema.StringAttribute{
				Optional: true,
			},
			"migrations_dir": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					common.DirValidator{},
				},
			},
			"version": schema.Int64Attribute{
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					common.VersionPlanModifier(),
				},
			},
			"target_version": schema.Int64Attribute{
				Optional: true,
			},
			"migrations": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					common.MigrationsPlanModifier(),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (y *ydbMigration) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating project resource")

	var plannedMigration ydbMigrationDataModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plannedMigration)...)

	createTimeout, timeoutInitError := plannedMigration.Timeouts.Create(ctx, common.DefaultTimeout)
	if timeoutInitError != nil {
		resp.Diagnostics.Append(timeoutInitError...)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	iamToken, err := y.providerConfig.SDK.CreateIAMToken(ctx)
	ctx = tflog.MaskMessageStrings(ctx, iamToken.IamToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create IAM token", err.Error())
		return
	}

	dbString := makeDbString(
		plannedMigration.Endpoint.ValueString(),
		plannedMigration.Database.ValueString(),
		iamToken.IamToken,
		plannedMigration.TlsEnabled.ValueBoolPointer(),
	)

	db, err := goose.OpenDBWithDriver("ydb", dbString)
	if err != nil {
		resp.Diagnostics.AddError("Failed to open DB", err.Error())
		return
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("goose: failed to close DB: %v\n", err)
		}
	}()

	if plannedMigration.MigrationTable.ValueString() != "" {
		goose.SetTableName(plannedMigration.MigrationTable.ValueString())
	}

	if err := goose.UpContext(ctx, db, plannedMigration.MigrationsDir.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to migrate", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plannedMigration)...)
}

func (y *ydbMigration) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Reading migration in db")
	var stateMigration ydbMigrationDataModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateMigration)...)

	iamToken, err := y.providerConfig.SDK.CreateIAMToken(ctx)
	ctx = tflog.MaskMessageStrings(ctx, iamToken.IamToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create IAM token", err.Error())
		tflog.Error(ctx, "Failed to create IAM token")
		return
	}

	dbString := makeDbString(
		stateMigration.Endpoint.ValueString(),
		stateMigration.Database.ValueString(),
		iamToken.IamToken,
		stateMigration.TlsEnabled.ValueBoolPointer(),
	)
	tflog.Info(ctx, fmt.Sprintf("DB string: %s", dbString))

	db, err := goose.OpenDBWithDriver("ydb", dbString)
	if err != nil {
		resp.Diagnostics.AddError("Failed to open DB", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to open DB %s", dbString))
		return
	}

	defer func() {
		if err := db.Close(); err != nil {
			resp.Diagnostics.AddError("Failed to close DB", err.Error())
		}
	}()

	current, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get current migration version", err.Error())
		tflog.Error(ctx, "Failed to get current migration version")
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Current version: %d", current))

	stateMigration.Version = types.Int64Value(current)

	resp.Diagnostics.Append(resp.State.Set(ctx, &stateMigration)...)
}

func (y *ydbMigration) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating migration resource")

	var planMigration, stateMigration ydbMigrationDataModel

	updateTimeout, timeoutInitError := planMigration.Timeouts.Update(ctx, utils.DefaultTimeout)
	if timeoutInitError != nil {
		resp.Diagnostics.Append(timeoutInitError...)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planMigration)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateMigration)...)

	if planMigration.MigrationTable.ValueString() != stateMigration.MigrationTable.ValueString() {
		resp.Diagnostics.AddError(
			"Cannot change migration_table",
			fmt.Sprintf("Migration table cannot be changed"),
		)
		return
	}

	iamToken, err := y.providerConfig.SDK.CreateIAMToken(ctx)
	ctx = tflog.MaskMessageStrings(ctx, iamToken.IamToken)
	if err != nil {
		return
	}

	dbString := makeDbString(
		stateMigration.Endpoint.ValueString(),
		stateMigration.Database.ValueString(),
		iamToken.IamToken,
		stateMigration.TlsEnabled.ValueBoolPointer(),
	)

	db, err := goose.OpenDBWithDriver("ydb", dbString)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("goose: failed to open DB: %v\n", err))
	}

	defer func() {
		if err := db.Close(); err != nil {
			tflog.Error(ctx, fmt.Sprintf("goose: failed to close DB: %v\n", err))
		}
	}()
	if planMigration.Version.ValueInt64() > stateMigration.Version.ValueInt64() {
		if err := goose.UpToContext(ctx, db, planMigration.MigrationsDir.ValueString(), planMigration.Version.ValueInt64()); err != nil {
			tflog.Error(ctx, fmt.Sprintf("goose up: %v", err))
		}
	} else if planMigration.Version.ValueInt64() < stateMigration.Version.ValueInt64() {
		if err := goose.DownToContext(ctx, db, planMigration.MigrationsDir.ValueString(), planMigration.Version.ValueInt64()); err != nil {
			tflog.Error(ctx, fmt.Sprintf("goose down: %v", err))
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &planMigration)...)
}

func (y *ydbMigration) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting migration resource")
	var stateMigration ydbMigrationDataModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateMigration)...)

	removeTimeout, timeoutInitError := stateMigration.Timeouts.Delete(ctx, utils.DefaultTimeout)
	if timeoutInitError != nil {
		resp.Diagnostics.Append(timeoutInitError...)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, removeTimeout)
	defer cancel()

	iamToken, err := y.providerConfig.SDK.CreateIAMToken(ctx)
	ctx = tflog.MaskMessageStrings(ctx, iamToken.IamToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create IAM token", err.Error())
	}

	dbString := makeDbString(
		stateMigration.Endpoint.ValueString(),
		stateMigration.Database.ValueString(),
		iamToken.IamToken,
		stateMigration.TlsEnabled.ValueBoolPointer(),
	)

	db, err := goose.OpenDBWithDriver("ydb", dbString)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("goose: failed to open DB: %v\n", err))
	}

	defer func() {
		if err := db.Close(); err != nil {
			tflog.Error(ctx, fmt.Sprintf("goose: failed to close DB: %v\n", err))
		}
	}()

	if err := goose.DownToContext(ctx, db, stateMigration.MigrationsDir.ValueString(), 0); err != nil {
		tflog.Error(ctx, fmt.Sprintf("goose down: %v", err))
	}

}

func (y *ydbMigration) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig, ok := req.ProviderData.(*provider_config.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *provider_config.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	y.providerConfig = providerConfig
}

func makeDbString(endpoint string, database string, token string, tlsEnabled *bool) string {
	tls := "grpcs"
	if tlsEnabled != nil && !*tlsEnabled {
		tls = "grpc"
	}
	options := map[string]string{
		"go_query_mode": "scripting",
		"go_fake_tx":    "scripting",
		"go_query_bind": "declare,numeric",
		"token":         token,
	}
	q := strings.Builder{}
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if q.Len() > 0 {
			q.WriteByte('&')
		}
		q.WriteString(k)
		q.WriteByte('=')
		q.WriteString(options[k])
	}

	return fmt.Sprintf("%s://%s%s?%s", tls, endpoint, database, q.String())
}
