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

package validator_test

import (
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

func helpSchemaFromTables(tables []config.TargetTable) *validator.DbSchema {
	return validator.NewDbSchema("public", tables)
}

func TestValidateSchema(t *testing.T) {
	t.Parallel()
	t.Run("happy_path", func(t *testing.T) {
		t.Parallel()
		test := func(name string, vecs []validator.Vec, schema *validator.DbSchema) {
			t.Helper()
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				t.Helper()
				err := validator.ValidateSchema(vecs, schema)
				require.NoError(t, err)
			})
		}

		test("both_args_nil", nil, nil)
		test("both_exists", []validator.Vec{
			{
				Op:   config.OpInsert,
				Tbl:  "tbl1",
				Cols: []string{"col1", "col2"},
			},
		}, helpSchemaFromTables([]config.TargetTable{
			{
				Table:  "tbl1",
				Fields: []string{"col1", "col2"},
			},
		}))
	})

	t.Run("bad_path", func(t *testing.T) {
		t.Parallel()
		test := func(name string, vecs []validator.Vec, schema *validator.DbSchema, err error) {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				err2 := validator.ValidateSchema(vecs, schema)
				require.Error(t, err2)
				require.ErrorIs(t, err2, err)
			})
		}

		test("tbl_not_exists", []validator.Vec{
			{
				Op:   config.OpInsert,
				Tbl:  "tbl1",
				Cols: []string{"col1", "col2"},
			},
		}, helpSchemaFromTables(nil), validator.ErrAccessDenied)
		test("col_not_exists", []validator.Vec{
			{
				Op:   config.OpInsert,
				Tbl:  "tbl1",
				Cols: []string{"col1", "col2"},
			},
		}, helpSchemaFromTables([]config.TargetTable{
			{
				Table:  "tbl1",
				Fields: []string{"col1"},
			},
		}), validator.ErrAccessDenied)
	})
}

func TestValidator(t *testing.T) {
	t.Parallel()
	target := config.Target{
		ID:            "t1",
		Description:   "",
		Tags:          nil,
		Type:          "postgres",
		Connection:    config.Connection{}, //nolint:exhaustruct
		DefaultSchema: "public",
		Tables: []config.TargetTable{
			{
				Table:  "public.clients",
				Fields: []string{"id", "name", "email"},
			},
		},
	}

	t.Run("bad_requests", func(t *testing.T) {
		t.Parallel()
		t.Run("query_has_no_statements", func(t *testing.T) {
			t.Parallel()
			query := ``
			err := validator.IsAllowed(nil, nil, query)
			require.Error(t, err)
		})

		t.Run("not_a_query", func(t *testing.T) {
			t.Parallel()
			query := `what time is it?`
			err := validator.IsAllowed(nil, nil, query)
			require.Error(t, err)
		})

		t.Run("query_has_several_statements", func(t *testing.T) {
			t.Parallel()
			query := `select 1; select 1`
			err := validator.IsAllowed(nil, nil, query)
			require.Error(t, err)
		})

		t.Run("star_select", func(t *testing.T) {
			t.Parallel()
			query := `select * from table`
			err := validator.IsAllowed(nil, nil, query)
			require.Error(t, err)
		})

		t.Run("schema_changes", func(t *testing.T) {
			t.Parallel()
			query := `create table aaa(id text);`
			err := validator.IsAllowed(nil, nil, query)
			require.Error(t, err)
		})

		t.Run("alter_table", func(t *testing.T) {
			t.Parallel()
			query := `alter table aaa add column id text default '';`
			err := validator.IsAllowed(nil, nil, query)
			require.Error(t, err)
		})
	})

	t.Run("some_cases", func(t *testing.T) {
		t.Parallel()
		t.Run("complicated_query", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return true }
			query := `WITH regional_sales AS (
    SELECT region, SUM(amount) AS total_sales
    FROM orders
    GROUP BY region
), top_regions AS (
    SELECT region
    FROM regional_sales
    WHERE total_sales > (SELECT SUM(total_sales)/10 FROM regional_sales)
)
SELECT region,
       product,
       SUM(quantity) AS product_units,
       SUM(amount) AS product_sales
FROM orders
WHERE region IN (SELECT region FROM top_regions)
GROUP BY region, product;`
			err := validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query)
			require.ErrorIs(t, err, validator.ErrComplicatedQuery)
		})
	})

	t.Run("select", func(t *testing.T) {
		t.Parallel()
		t.Run("simple_allowed", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return true }
			query := `select id, name from clients;`
			err := validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query)
			require.NoError(t, err)
		})

		t.Run("simple_denied", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return false }
			query := `select id, name from clients;`
			err := validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})

		t.Run("select_from_allowed_select__is_not_allowed", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return true }
			query := `select id, name from (select id, name from clients)`
			err := validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query)
			// TODO: make it allowed. Actually this is legal query for this ACL.
			require.ErrorIs(t, err, validator.ErrComplicatedQuery)
		})
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()
		t.Run("simple_allowed", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return true }
			query := `update clients set id=1, name='john'`
			require.NoError(t, validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query))
		})
		t.Run("simple_denied", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return false }
			query := `update clients set id=1, name='john'`
			err := validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})
	})

	t.Run("delete", func(t *testing.T) {
		t.Parallel()
		t.Run("simple_allowed", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return true }
			query := `delete from clients where id=42`
			require.NoError(t, validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query))
		})
		t.Run("simple_denied", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return false }
			query := `delete from clients where id=42`
			err := validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})
	})

	t.Run("insert", func(t *testing.T) {
		t.Parallel()
		t.Run("simple_allowed", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return true }
			query := `insert into clients(id) values (42)`
			err := validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query)
			require.NoError(t, err)
		})

		t.Run("simple_denied", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return false }
			query := `insert into clients(id) values (42)`
			err := validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})

		t.Run("allowed_2", func(t *testing.T) {
			t.Parallel()
			haveAccess := func(_ validator.Vec) bool { return true }
			query := `insert into clients(id, name, email) values('11', '22', '33')`
			err := validator.IsAllowed(helpSchemaFromTables(target.Tables), haveAccess, query)
			require.NoError(t, err)
		})
	})
}
