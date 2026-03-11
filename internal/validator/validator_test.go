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

func testTargetTables() []config.TargetTable {
	return []config.TargetTable{
		{
			Table:  "public.clients",
			Fields: []string{"id", "name", "email"},
		},
	}
}

func TestValidatorBadRequests(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		query string
	}{
		{name: "query_has_no_statements", query: ``},
		{name: "not_a_query", query: `what time is it?`},
		{name: "query_has_several_statements", query: `select 1; select 1`},
		{name: "star_select", query: `select * from table`}, //nolint:unqueryvet
		{name: "schema_changes", query: `create table aaa(id text);`},
		{name: "alter_table", query: `alter table aaa add column id text default '';`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validator.IsAllowed(nil, nil, tc.query)
			require.Error(t, err)
		})
	}
}

func TestValidatorComplicatedQueries(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		query string
	}{
		{
			name: "complicated_query",
			query: `WITH regional_sales AS (
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
GROUP BY region, product;`,
		},
		{
			name: "cte_and_nested_subquery",
			query: `WITH regional_sales AS (
    SELECT region, SUM(amount) AS total_sales
    FROM orders
    GROUP BY region
)
SELECT region
FROM regional_sales
WHERE total_sales > (SELECT SUM(total_sales) / 10 FROM regional_sales)`,
		},
		{name: "join_using_clause", query: `SELECT c.id FROM clients AS c JOIN orders AS o USING (id)`},
		{name: "natural_join", query: `SELECT c.id FROM clients AS c NATURAL JOIN orders AS o`},
		{name: "join_subquery_rhs", query: `SELECT c.id FROM clients AS c JOIN (SELECT client_id FROM orders) AS o ON c.id = o.client_id`},
		{name: "union", query: `SELECT id FROM clients UNION SELECT id FROM orders`},
	}

	haveAccess := func(_ validator.Vec) bool { return true }
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, tc.query)
			require.ErrorIs(t, err, validator.ErrComplicatedQuery)
		})
	}
}

func TestValidatorSelect(t *testing.T) {
	t.Parallel()

	t.Run("simple_allowed", func(t *testing.T) {
		t.Parallel()
		haveAccess := func(_ validator.Vec) bool { return true }
		query := `select id, name from clients;`
		err := validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, query)
		require.NoError(t, err)
	})

	t.Run("simple_denied", func(t *testing.T) {
		t.Parallel()
		haveAccess := func(_ validator.Vec) bool { return false }
		query := `select id, name from clients;`
		err := validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, query)
		require.ErrorIs(t, err, validator.ErrAccessDenied)
	})

	t.Run("select_from_allowed_select__is_not_allowed", func(t *testing.T) {
		t.Parallel()
		haveAccess := func(_ validator.Vec) bool { return true }
		query := `select id, name from (select id, name from clients)`
		err := validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, query)
		require.ErrorIs(t, err, validator.ErrComplicatedQuery)
	})

	t.Run("join_access_requires_all_tables", func(t *testing.T) {
		t.Parallel()

		schema := helpSchemaFromTables([]config.TargetTable{
			{
				Table:  "public.clients",
				Fields: []string{"id", "name"},
			},
			{
				Table:  "public.orders",
				Fields: []string{"id", "client_id", "status"},
			},
		})

		haveAccess := func(vec validator.Vec) bool {
			return vec.Tbl == "public.clients"
		}

		query := `select c.id, o.id
from public.clients as c
join public.orders as o on c.id = o.client_id
where o.status = 'paid'`
		err := validator.IsAllowed(schema, haveAccess, query)
		require.ErrorIs(t, err, validator.ErrAccessDenied)
	})

	t.Run("join_schema_checks_every_referenced_table", func(t *testing.T) {
		t.Parallel()

		schema := helpSchemaFromTables([]config.TargetTable{
			{
				Table:  "public.clients",
				Fields: []string{"id", "name"},
			},
		})

		haveAccess := func(_ validator.Vec) bool { return true }
		query := `select c.id, o.id
from public.clients as c
join public.orders as o on c.id = o.client_id`
		err := validator.IsAllowed(schema, haveAccess, query)
		require.ErrorIs(t, err, validator.ErrAccessDenied)
	})
}

func TestValidatorUpdate(t *testing.T) {
	t.Parallel()

	t.Run("simple_allowed", func(t *testing.T) {
		t.Parallel()
		haveAccess := func(_ validator.Vec) bool { return true }
		query := `update clients set id=1, name='john'`
		require.NoError(t, validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, query))
	})

	t.Run("simple_denied", func(t *testing.T) {
		t.Parallel()
		haveAccess := func(_ validator.Vec) bool { return false }
		query := `update clients set id=1, name='john'`
		err := validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, query)
		require.ErrorIs(t, err, validator.ErrAccessDenied)
	})
}

func TestValidatorDelete(t *testing.T) {
	t.Parallel()

	t.Run("simple_allowed", func(t *testing.T) {
		t.Parallel()
		haveAccess := func(_ validator.Vec) bool { return true }
		query := `delete from clients where id=42`
		require.NoError(t, validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, query))
	})

	t.Run("simple_denied", func(t *testing.T) {
		t.Parallel()
		haveAccess := func(_ validator.Vec) bool { return false }
		query := `delete from clients where id=42`
		err := validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, query)
		require.ErrorIs(t, err, validator.ErrAccessDenied)
	})
}

func TestValidatorInsert(t *testing.T) {
	t.Parallel()

	t.Run("simple_allowed", func(t *testing.T) {
		t.Parallel()
		haveAccess := func(_ validator.Vec) bool { return true }
		query := `insert into clients(id) values (42)`
		err := validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, query)
		require.NoError(t, err)
	})

	t.Run("simple_denied", func(t *testing.T) {
		t.Parallel()
		haveAccess := func(_ validator.Vec) bool { return false }
		query := `insert into clients(id) values (42)`
		err := validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, query)
		require.ErrorIs(t, err, validator.ErrAccessDenied)
	})

	t.Run("allowed_2", func(t *testing.T) {
		t.Parallel()
		haveAccess := func(_ validator.Vec) bool { return true }
		query := `insert into clients(id, name, email) values('11', '22', '33')`
		err := validator.IsAllowed(helpSchemaFromTables(testTargetTables()), haveAccess, query)
		require.NoError(t, err)
	})
}
