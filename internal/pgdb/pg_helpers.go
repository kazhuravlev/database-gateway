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

package pgdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/just"
)

func BuildDBDsn(cfg config.PostgresConfig) string { //nolint:gocritic
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.Database,
		just.If(cfg.UseSSL, "prefer", "disable"),
	)
}

func ConnectToPg(cfg config.PostgresConfig) (*sql.DB, error) { //nolint:gocritic
	postgresDSN := BuildDBDsn(cfg)
	dbConn, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	dbConn.SetMaxIdleConns(2)                  //nolint:mnd
	dbConn.SetConnMaxLifetime(5 * time.Minute) //nolint:mnd
	dbConn.SetMaxOpenConns(cfg.MaxPoolSize)

	if err := dbConn.PingContext(context.TODO()); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return dbConn, nil
}
