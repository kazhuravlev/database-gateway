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

	"github.com/kazhuravlev/database-gateway/internal/app"
	"github.com/kazhuravlev/database-gateway/internal/app/rules"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/migrator"
	"github.com/kazhuravlev/database-gateway/internal/pgdb"
	"github.com/kazhuravlev/database-gateway/internal/storage"
	"github.com/kazhuravlev/database-gateway/internal/storage/migrations"
	"github.com/kazhuravlev/just"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func withConfig(action func(c *cli.Context, cfg config.Config) error) cli.ActionFunc {
	return func(c *cli.Context) error {
		configFilename := c.String(keyConfig)

		cfg, err := just.JsonParseTypeF[config.Config](configFilename)
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}

		return action(c, *cfg)
	}
}

func newMigrator(cfg config.PostgresConfig) (*migrator.Migrator, error) {
	dbConn, err := pgdb.ConnectToPg(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "connect to postgres")
	}

	migratorInst, err := migrator.New(migrator.NewOptions(
		migrations.Migrations,
		migrations.TableName,
		migrations.AbsMigrationsDir,
		dbConn,
	))
	if err != nil {
		return nil, errors.Wrap(err, "create new migrator")
	}

	return migratorInst, nil
}

func withApp(cmd func(context.Context, *cli.Context, config.Config, *app.Service, *slog.Logger) error) func(*cli.Context, config.Config) error {
	return func(c *cli.Context, cfg config.Config) error {
		ctx, cancel := context.WithCancel(c.Context)
		defer cancel()

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource:   false,
			Level:       slog.LevelInfo,
			ReplaceAttr: nil,
		}))

		logger.Info("start")

		migratorInst, err := newMigrator(cfg.Storage)
		if err != nil {
			return fmt.Errorf("create new migrator: %w", err)
		}

		if err := migratorInst.Up(); err != nil {
			return fmt.Errorf("up all migrations: %w", err)
		}

		dbConnWrite, err := pgdb.ConnectToPg(cfg.Storage)
		if err != nil {
			return fmt.Errorf("connect to db: %w", err)
		}

		storageInst, err := storage.New(storage.NewOptions(logger, dbConnWrite))
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}

		appInst, err := app.New(app.NewOptions(logger, cfg.Targets, cfg.Users, rules.New(cfg.ACLs), storageInst))
		if err != nil {
			return fmt.Errorf("create app instance: %w", err)
		}

		return cmd(ctx, c, cfg, appInst, logger)
	}
}
