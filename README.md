# Goose Migration Provider

Example usage:

```hcl
resource "yandex_ydb_database_serverless" "db" {
  name      = "db"
  folder_id = var.folder_id
}

resource "goose_ydb_migration" "db" {
  depends_on = [yandex_ydb_database_serverless.db]
  endpoint   = yandex_ydb_database_serverless.db.ydb_api_endpoint
  database   = yandex_ydb_database_serverless.db.database_path
  migrations_dir = "migrations"
}
```

Assuming the following directory structure:

```
.
├── main.tf
└── migrations
    ├── 01_orders.sql
    └── 02_payments.sql
```

Plan example:

```hcl
  # goose_ydb_migration.db will be created
  + resource "goose_ydb_migration" "db" {
      + database       = "/ru-central1/b1g***/etn**"
      + endpoint       = "ydb.serverless.yandexcloud.net:2135"
      + migrations     = [
          + "migrations/01_orders.sql",
          + "migrations/02_payments.sql",
        ]
      + migrations_dir = "migrations"
      + version        = 2
    }
```

It's also possible to define `target_version`:

```
resource "goose_ydb_migration" "db" {
  depends_on = [yandex_ydb_database_serverless.db]
  endpoint   = yandex_ydb_database_serverless.db.ydb_api_endpoint
  database   = yandex_ydb_database_serverless.db.database_path
  migrations_dir = "migrations"
  target_version = 1
}
```

In this case, the migration will be applied to the target version upwards from below:
```hcl
  # goose_ydb_migration.db will be created
  + resource "goose_ydb_migration" "db" {
      + database       = "/ru-central1/b1g***/etn**"
      + endpoint       = "ydb.serverless.yandexcloud.net:2135"
      + migrations     = [
          + "migrations/01_orders.sql",
        ]
      + migrations_dir = "migrations"
      + target_version = 1
      + version        = 1
    }
```

or downwards from above:
```hcl
  # goose_ydb_migration.db will be updated in-place
  ~ resource "goose_ydb_migration" "db" {
    ~ migrations     = [
        "migrations/01_orders.sql",
      - "migrations/02_payments.sql",
      ]
    + target_version = 1
    ~ version        = 2 -> 1
    # (3 unchanged attributes hidden)
    }
```