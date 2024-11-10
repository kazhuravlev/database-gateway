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
)

type conn struct {
	ctx context.Context //nolint:containedctx // because it is part of transaction
	db  *sql.DB
}

func NewConn(ctx context.Context, db *sql.DB) *conn {
	return &conn{ctx: ctx, db: db}
}

func (c *conn) Exec(query string, args ...interface{}) (sql.Result, error) {
	return c.db.Exec(query, args...) //nolint:wrapcheck
}

func (c *conn) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return c.db.ExecContext(ctx, query, args...) //nolint:wrapcheck
}

func (c *conn) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.Query(query, args...) //nolint:wrapcheck
}

func (c *conn) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...) //nolint:wrapcheck
}

func (c *conn) Context() context.Context {
	return c.ctx
}

type tx struct {
	ctx context.Context //nolint:containedctx // because it is part of transaction
	db  *sql.Tx
}

func NewTx(ctx context.Context, db *sql.Tx) *tx {
	return &tx{ctx: ctx, db: db}
}

func (t *tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	return t.db.ExecContext(t.ctx, query, args...) //nolint:wrapcheck
}

func (t *tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.db.ExecContext(ctx, query, args...) //nolint:wrapcheck
}

func (t *tx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return t.db.QueryContext(t.ctx, query, args...) //nolint:wrapcheck
}

func (t *tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.db.QueryContext(ctx, query, args...) //nolint:wrapcheck
}

func (t *tx) Context() context.Context {
	return t.ctx
}
