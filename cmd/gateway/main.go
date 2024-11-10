// Database Gateway provides access to servers with ACL for safe and restricted database interactions.
// Copyright (C) 2024  Kirill Zhuravlev
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kazhuravlev/database-gateway/internal/uuid6"

	"github.com/go-jet/jet/v2/generator/metadata"
	"github.com/go-jet/jet/v2/generator/postgres"
	"github.com/go-jet/jet/v2/generator/template"
	postgres2 "github.com/go-jet/jet/v2/postgres"
	"github.com/kazhuravlev/database-gateway/internal/app"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/facade"
	"github.com/kazhuravlev/database-gateway/internal/pgdb"
	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"
)

const keyConfig = "config"

func main() {
	application := &cli.App{ //nolint:exhaustruct
		Name: "dbgw",
		Flags: []cli.Flag{
			&cli.StringFlag{ //nolint:exhaustruct
				Aliases: []string{"c"},
				Name:    keyConfig,
				Value:   "config.json",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "run",
				Action: withConfig(withApp(cmdRun)),
			},
			{
				Name:   "jet-generate",
				Action: withConfig(cmdGenerateModels),
			},
			{
				Name:        "migrate-up",
				Description: "Run all migrations up",
				Action:      withConfig(cmdMigrateUp),
			},
			{
				Name:        "migrate-down-one",
				Description: "Rollback migration to down one-by-one",
				Action:      withConfig(cmdMigrateDownOne),
			},
			{
				Name:        "migrate-new",
				Description: "Create new migration file",
				Action:      withConfig(cmdMigrateCreateNew),
			},
		},
	}

	if err := application.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func cmdRun(ctx context.Context, c *cli.Context, cfg config.Config, appInst *app.Service, logger *slog.Logger) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	fInst, err := facade.New(facade.NewOptions(logger, appInst, cfg.Facade.CookieSecret, cfg.Facade.Port))
	if err != nil {
		return fmt.Errorf("create facade: %w", err)
	}

	if err := fInst.Run(ctx); err != nil {
		return fmt.Errorf("run facade: %w", err)
	}

	return nil
}

func cmdGenerateModels(c *cli.Context, cfg config.Config) error {
	// map[TABLE_NAME]map[FIELD_NAME]FIELD_TYPE
	customFields := map[string]map[string]template.Type{
		"query_results": {
			"id":       template.NewType(uuid6.Nil()),
			"user_id":  template.NewType(config.UserID("")),
			"response": template.NewType([]byte{}),
		},
	}

	postgresDSN := pgdb.BuildDbDsn(cfg.Storage)

	dbTemplate := template.Default(postgres2.Dialect).
		UseSchema(func(schema metadata.Schema) template.Schema {
			return template.DefaultSchema(schema).
				// UsePath("../").
				UseModel(template.DefaultModel().
					UseTable(func(table metadata.Table) template.TableModel {
						return template.DefaultTableModel(table).
							UseField(func(column metadata.Column) template.TableModelField {
								defaultTableModelField := template.DefaultTableModelField(column)

								customType, ok := customFields[table.Name][column.Name]
								if ok {
									defaultTableModelField.Type = customType
								}

								return defaultTableModelField
							})
					}),
				)
		})

	if err := postgres.GenerateDSN(postgresDSN, "public", "./internal/storage/jetgen", dbTemplate); err != nil {
		return fmt.Errorf("generate jet models: %w", err)
	}

	return nil
}

func cmdMigrateUp(c *cli.Context, cfg config.Config) error {
	migratorInst, err := newMigrator(cfg.Storage)
	if err != nil {
		return fmt.Errorf("create new migrator: %w", err)
	}

	if err := migratorInst.Up(); err != nil {
		return fmt.Errorf("up all migrations: %w", err)
	}

	return nil
}

func cmdMigrateDownOne(c *cli.Context, cfg config.Config) error {
	migratorInst, err := newMigrator(cfg.Storage)
	if err != nil {
		return fmt.Errorf("create new migrator: %w", err)
	}

	if err := migratorInst.DownOne(); err != nil {
		return fmt.Errorf("down one migration: %w", err)
	}

	return nil
}

func cmdMigrateCreateNew(c *cli.Context, cfg config.Config) error {
	migratorInst, err := newMigrator(cfg.Storage)
	if err != nil {
		return fmt.Errorf("create new migrator: %w", err)
	}

	if err := migratorInst.CreateNewMigration("unnamed_migration", "sql"); err != nil {
		return fmt.Errorf("create new migration: %w", err)
	}

	return nil
}
