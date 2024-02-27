package common

import (
	"context"
	"fmt"
	"math"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pressly/goose/v3"
)

const maxVersion = math.MaxInt64

type versionPlanModifier struct{}

func (v versionPlanModifier) Description(_ context.Context) string {
	return "Calculates the version for the migration"
}

func (v versionPlanModifier) MarkdownDescription(_ context.Context) string {
	return "Calculates the version for the migration"
}

func (m versionPlanModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {

	var migrationsDir string
	var target *int64

	diags := req.Plan.GetAttribute(ctx, path.Root("migrations_dir"), &migrationsDir)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = req.Plan.GetAttribute(ctx, path.Root("target_version"), &target)

	if resp.Diagnostics.HasError() {
		return
	}
	if migrationsDir == "" {
		tflog.Debug(ctx, "no migrations_dir in plan")
		req.State.GetAttribute(ctx, path.Root("migrations_dir"), &migrationsDir)
	}

	if migrationsDir == "" {
		resp.Diagnostics.AddError("migrations_dir", "migrations_dir is required")
		return
	}

	migrations, err := goose.CollectMigrations(
		migrationsDir,
		0,
		maxVersion,
	)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to collect migrations: %s %s", migrationsDir, err.Error()))
		return
	}

	if len(migrations) > 0 {
		maxKnownVersion := migrations[len(migrations)-1].Version
		if target == nil {
			resp.PlanValue = types.Int64Value(maxKnownVersion)
			return
		}
		if *target < 0 {
			resp.Diagnostics.AddError("target_version", "target_version is less than 0")
			return
		}
		if *target > maxKnownVersion {
			resp.Diagnostics.AddError("target_version", "target_version is greater than the latest migration")
			return
		} else {
			resp.PlanValue = types.Int64Value(*target)
		}
	}

}

func VersionPlanModifier() planmodifier.Int64 {
	return versionPlanModifier{}
}

type migrationsPlanModifier struct{}

func (m migrationsPlanModifier) Description(_ context.Context) string {
	return "Calculates the migration list"
}

func (m migrationsPlanModifier) MarkdownDescription(_ context.Context) string {
	return "Calculates the migration list"
}

func (m migrationsPlanModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	var migrationsDir string
	var target *int64

	diags := req.Plan.GetAttribute(ctx, path.Root("migrations_dir"), &migrationsDir)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = req.Plan.GetAttribute(ctx, path.Root("target_version"), &target)

	if resp.Diagnostics.HasError() {
		return
	}
	if migrationsDir == "" {
		tflog.Debug(ctx, "no migrations_dir in plan")
		req.State.GetAttribute(ctx, path.Root("migrations_dir"), &migrationsDir)
	}

	if migrationsDir == "" {
		resp.Diagnostics.AddError("migrations_dir", "migrations_dir is required")
		return
	}

	migrations, err := goose.CollectMigrations(
		migrationsDir,
		0,
		maxVersion,
	)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to collect migrations: %s %s", migrationsDir, err.Error()))
		return
	}

	var migrationVersions []string
	for _, migration := range migrations {
		if target != nil && migration.Version > *target {
			continue
		}
		migrationVersions = append(migrationVersions, migration.String())
	}
	val, diag := types.ListValueFrom(ctx, types.StringType, migrationVersions)
	resp.Diagnostics.Append(diag...)
	resp.PlanValue = val
}

func MigrationsPlanModifier() planmodifier.List {
	return migrationsPlanModifier{}
}
