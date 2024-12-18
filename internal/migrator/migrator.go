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

package migrator

import (
	"fmt"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

const dialect = "postgres"

type Migrator struct {
	opts Options
}

func New(opts Options) (*Migrator, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	goose.SetBaseFS(opts.migrationsFs)
	goose.SetTableName(opts.migrationsTableName)
	goose.SetSequential(true)

	if err := goose.SetDialect(dialect); err != nil {
		return nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	return &Migrator{
		opts: opts,
	}, nil
}

func (m *Migrator) CreateNewMigration(name, typ string) error {
	if err := goose.Create(m.opts.db, m.opts.migrationsDir, name, typ); err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	return nil
}

func (m *Migrator) Up() error {
	if err := goose.Up(m.opts.db, "."); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

func (m *Migrator) DownOne() error {
	if err := goose.Down(m.opts.db, "."); err != nil {
		return fmt.Errorf("migrate down: %w", err)
	}

	return nil
}

func (m *Migrator) DownAll() error {
	if err := goose.DownTo(m.opts.db, ".", 0); err != nil {
		return fmt.Errorf("migrate down one: %w", err)
	}

	return nil
}
