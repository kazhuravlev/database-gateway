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

package app

import (
	"log/slog"

	"github.com/kazhuravlev/database-gateway/internal/app/rules"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/storage"
)

//go:generate toolset run options-gen -from-struct=Options
type Options struct {
	logger  *slog.Logger       `option:"mandatory" validate:"required"`
	targets []config.Target    `option:"mandatory" validate:"required"`
	users   config.UsersConfig `option:"mandatory" validate:"required"`
	acls    *rules.ACLs        `option:"mandatory" validate:"required"`
	storage *storage.Service   `option:"mandatory" validate:"required"`
}
