package goose_ydb_migration

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ydbMigrationDataModel struct {
	Endpoint       types.String   `tfsdk:"endpoint"`
	Database       types.String   `tfsdk:"database"`
	TlsEnabled     types.Bool     `tfsdk:"tls_enabled"`
	MigrationTable types.String   `tfsdk:"migration_table"`
	MigrationsDir  types.String   `tfsdk:"migrations_dir"`
	Version        types.Int64    `tfsdk:"version"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}
