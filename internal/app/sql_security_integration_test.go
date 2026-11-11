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

package app //nolint:testpackage // Exercises the actual internal execution boundary and connection settings.

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

// Runs against an isolated PostgreSQL service in CI. Never points at configured
// application targets; all fixtures live in uniquely named disposable schemas.
func TestAuthorizedSQLPostgres(t *testing.T) { //nolint:funlen // One isolated database fixture verifies the complete authorization boundary.
	t.Parallel()
	dsn := os.Getenv("DBGW_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("DBGW_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = admin.Close(context.Background()) })
	safe := "dbgw_safe_" + uuid.New().String()[:8]
	evil := "dbgw_evil_" + uuid.New().String()[:8]
	exec := func(sql string) { t.Helper(); _, err := admin.Exec(ctx, sql); require.NoError(t, err) }
	exec("CREATE SCHEMA " + safe)
	t.Cleanup(func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+safe+" CASCADE")
		require.NoError(t, err)
	})
	exec("CREATE SCHEMA " + evil)
	t.Cleanup(func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+evil+" CASCADE")
		require.NoError(t, err)
	})
	exec("CREATE TABLE " + safe + ".clients(secret text, id integer PRIMARY KEY, name varchar)")
	exec("CREATE TABLE " + evil + ".clients(id integer, name varchar)")
	exec("INSERT INTO " + safe + ".clients VALUES('private',1,'ALICE')")
	exec("INSERT INTO " + evil + ".clients VALUES(999,'WRONG TABLE')")
	exec("CREATE FUNCTION " + evil + ".lower(varchar) RETURNS text LANGUAGE sql AS $$ SELECT 'HIJACKED'::text $$")
	exec("SET search_path=" + evil + "," + safe + ",pg_catalog")
	var shadowID int
	require.NoError(t, admin.QueryRow(ctx, "SELECT id FROM clients").Scan(&shadowID))
	require.Equal(t, 999, shadowID)
	var shadowFunction string
	require.NoError(t, admin.QueryRow(ctx, "SELECT lower(name) FROM "+safe+".clients").Scan(&shadowFunction))
	require.Equal(t, "HIJACKED", shadowFunction)
	var renamedSecret string
	require.NoError(t, admin.QueryRow(ctx, "SELECT id FROM "+safe+".clients c(id,other_id)").Scan(&renamedSecret))
	require.Equal(t, "private", renamedSecret)

	connection, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	target := config.Target{ID: "integration", DefaultSchema: safe, Connection: config.Connection{
		Host: connection.Host, Port: int(connection.Port), User: connection.User, Password: connection.Password, DB: connection.Database,
	}}
	schema := validator.NewDbSchema(safe, []config.TargetTable{{Table: safe + ".clients", Fields: []string{"id", "name"}}})
	service := &Service{opts: Options{logger: slog.New(slog.DiscardHandler)}, connsMu: new(sync.RWMutex), conns: make(map[config.TargetID]*pgxpool.Pool)}
	pool, err := service.getConnection(ctx, target)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var searchPath, stringsMode, encoding string
	require.NoError(t, pool.QueryRow(ctx, "SHOW search_path").Scan(&searchPath))
	require.NoError(t, pool.QueryRow(ctx, "SHOW standard_conforming_strings").Scan(&stringsMode))
	require.NoError(t, pool.QueryRow(ctx, "SHOW client_encoding").Scan(&encoding))
	require.Equal(t, "pg_catalog, pg_temp", searchPath)
	require.Equal(t, "on", stringsMode)
	require.Equal(t, "UTF8", encoding)

	for _, query := range []string{"SELECT id, lower(name) FROM clients", "SELECT c.id, lower(c.name) FROM clients c"} {
		authorized, err := validator.Authorize(schema, func(validator.Vec) bool { return true }, query)
		require.NoError(t, err)
		result, err := service.execQuery(ctx, &target, authorized)
		require.NoError(t, err)
		require.Equal(t, [][]string{{"1", "alice"}}, result.Rows)
	}
	// The same statement is legal in PostgreSQL but cannot read a renamed secret.
	for _, query := range []string{
		"SELECT id FROM clients c(id,other_id)",
		"UPDATE clients SET name='bob' RETURNING secret AS id",
		"INSERT INTO clients(id,name) VALUES(1,'bob') ON CONFLICT(id) DO UPDATE SET name=secret",
	} {
		authorized, err := validator.Authorize(schema, func(validator.Vec) bool { return true }, query)
		require.Error(t, err)
		require.Empty(t, authorized.SQL())
	}
	_, err = service.execQuery(ctx, &target, validator.AuthorizedQuery{})
	require.ErrorIs(t, err, ErrForbidden)
	authorized, err := validator.Authorize(schema, func(validator.Vec) bool { return true }, "UPDATE clients SET name='BOB' WHERE id=1 RETURNING id,name")
	require.NoError(t, err)
	result, err := service.execQuery(ctx, &target, authorized)
	require.NoError(t, err)
	require.Equal(t, [][]string{{"1", "BOB"}}, result.Rows)
}
