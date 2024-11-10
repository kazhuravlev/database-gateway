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

package migrations

import (
	"embed"
	"path/filepath"
)

//go:embed *.sql
var Migrations embed.FS

// AbsMigrationsDir должен указывать на директорию, где хранятся миграции.
var AbsMigrationsDir = func() string { //nolint:gochecknoglobals
	abs, err := filepath.Abs(absMigrationsDir)
	if err != nil {
		panic("the sky is falling")
	}

	return abs
}()

const (
	absMigrationsDir = "./internal/storage/migrations"
	TableName        = "goose_migrations"
)
