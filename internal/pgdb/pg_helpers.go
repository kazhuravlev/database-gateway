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
	"database/sql"
	"fmt"
	"time"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/just"
	"github.com/pkg/errors"
)

func BuildDbDsn(cfg config.PostgresConfig) string {
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

func ConnectToPg(cfg config.PostgresConfig) (*sql.DB, error) {
	postgresDSN := BuildDbDsn(cfg)
	dbConn, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		return nil, errors.Wrap(err, "connect to postgres")
	}

	dbConn.SetMaxIdleConns(2)                  //nolint:gomnd // it is obvious
	dbConn.SetConnMaxLifetime(5 * time.Minute) //nolint:gomnd // it is obvious
	dbConn.SetMaxOpenConns(cfg.MaxPoolSize)

	if err := dbConn.Ping(); err != nil {
		return nil, errors.Wrap(err, "ping postgres")
	}

	return dbConn, nil
}
