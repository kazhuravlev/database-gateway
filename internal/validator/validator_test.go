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

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	t.Parallel()
	target := config.Target{
		ID:         "t1",
		Type:       "postgres",
		Connection: config.Connection{}, //nolint:exhaustruct
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
			acls := []config.ACL{{
				Op:     config.OpSelect,
				Target: "t1",
				Tbl:    "public.clients",
				Allow:  true,
			}}
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
			err := validator.IsAllowed(target.Tables, acls, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})
	})

	t.Run("select", func(t *testing.T) {
		t.Parallel()
		t.Run("simple_allowed", func(t *testing.T) {
			t.Parallel()
			acls := []config.ACL{{
				Op:     config.OpSelect,
				Target: "t1",
				Tbl:    "public.clients",
				Allow:  true,
			}}
			query := `select id, name from clients;`
			err := validator.IsAllowed(target.Tables, acls, query)
			require.NoError(t, err)
		})

		t.Run("simple_denied", func(t *testing.T) {
			t.Parallel()
			acls := []config.ACL{{
				Op:     config.OpSelect,
				Target: "t1",
				Tbl:    "public.clients",
				Allow:  false,
			}}
			query := `select id, name from clients;`
			err := validator.IsAllowed(target.Tables, acls, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})

		t.Run("select_from_allowed_select__is_not_allowed", func(t *testing.T) {
			t.Parallel()
			acls := []config.ACL{{
				Op:     config.OpSelect,
				Target: "t1",
				Tbl:    "public.clients",
				Allow:  true,
			}}
			query := `select id, name from (select id, name from clients)`
			err := validator.IsAllowed(target.Tables, acls, query)
			// TODO: make it allowed. Actually this is legal query for this ACL.
			require.ErrorIs(t, err, validator.ErrComplicatedQuery)
		})
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()
		t.Run("simple_allowed", func(t *testing.T) {
			t.Parallel()
			acls := []config.ACL{{
				Op:     config.OpUpdate,
				Target: "t1",
				Tbl:    "public.clients",
				Allow:  true,
			}}
			query := `update clients set id=1 and name='john'`
			err := validator.IsAllowed(target.Tables, acls, query)
			require.NoError(t, err)
		})
		t.Run("simple_denied", func(t *testing.T) {
			t.Parallel()
			acls := []config.ACL{{
				Op:     config.OpUpdate,
				Target: "t1",
				Tbl:    "public.clients",
				Allow:  false,
			}}
			query := `update clients set id=1 and name='john'`
			err := validator.IsAllowed(target.Tables, acls, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})
	})

	t.Run("delete", func(t *testing.T) {
		t.Parallel()
		t.Run("simple_allowed", func(t *testing.T) {
			t.Parallel()
			acls := []config.ACL{{
				Op:     config.OpDelete,
				Target: "t1",
				Tbl:    "public.clients",
				Allow:  true,
			}}
			query := `delete from clients where id=42`
			err := validator.IsAllowed(target.Tables, acls, query)
			require.NoError(t, err)
		})
		t.Run("simple_denied", func(t *testing.T) {
			t.Parallel()
			acls := []config.ACL{{
				Op:     config.OpDelete,
				Target: "t1",
				Tbl:    "public.clients",
				Allow:  false,
			}}
			query := `delete from clients where id=42`
			err := validator.IsAllowed(target.Tables, acls, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})
	})

	t.Run("insert", func(t *testing.T) {
		t.Parallel()
		t.Run("simple_allowed", func(t *testing.T) {
			t.Parallel()
			acls := []config.ACL{{
				Op:     config.OpInsert,
				Target: "t1",
				Tbl:    "public.clients",
				Allow:  true,
			}}
			query := `insert into clients(id) values (42)`
			err := validator.IsAllowed(target.Tables, acls, query)
			require.NoError(t, err)
		})

		t.Run("simple_denied", func(t *testing.T) {
			t.Parallel()
			acls := []config.ACL{{
				Op:     config.OpInsert,
				Target: "t1",
				Tbl:    "public.clients",
				Allow:  false,
			}}
			query := `insert into clients(id) values (42)`
			err := validator.IsAllowed(target.Tables, acls, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})
	})
}

func TestVector(t *testing.T) {
	t.Parallel()
	req := &tree.Select{
		With: nil,
		Select: &tree.SelectClause{
			Distinct: false,
			DistinctOn: tree.DistinctOn{
				tree.NewUnresolvedName("distinct_col"),
			},
			Exprs: tree.SelectExprs{
				{
					Expr: tree.NewUnresolvedName("col_1"),
					As:   "alias_1",
				},
				{
					Expr: tree.NewUnresolvedName("col_2"),
					As:   "alias_2",
				},
			},
			From: tree.From{
				Tables: tree.TableExprs{
					tree.NewTableName("", "clients"),
				},
				AsOf: tree.AsOfClause{
					Expr: nil,
				},
			},
			Where: &tree.Where{
				Type: "",
				Expr: tree.NewUnresolvedName("where_1"),
			},
			GroupBy: tree.GroupBy{
				tree.NewUnresolvedName("group_1"),
			},
			Having: &tree.Where{
				Type: "",
				Expr: tree.NewUnresolvedName("having_1"),
			},
			Window: tree.Window{
				{
					Name:    "",
					RefName: "",
					Partitions: tree.Exprs{
						tree.NewUnresolvedName("part_1"),
					},
					OrderBy: tree.OrderBy{
						{
							Expr: tree.NewUnresolvedName("part_order_col"),
						},
					},
					Frame: nil,
				},
			},
			TableSelect: false,
		},
		OrderBy: tree.OrderBy{
			{
				Expr: tree.NewUnresolvedName("order_col"),
			},
		},
		Limit:   nil,
		Locking: nil,
	}
	vec, err := validator.MakeSelectVec(req)
	require.NoError(t, err)
	expVec := validator.VecSelect{
		Tbl: "public.clients",
		Cols: []string{
			"col_1",
			"col_2",
			"distinct_col",
			"group_1",
			"having_1",
			"order_col",
			"part_1",
			"part_order_col",
			"where_1",
		},
	}
	require.Equal(t, expVec, *vec)

	res, err := validator.GetColumnNames(req)
	require.NoError(t, err)
	require.Equal(t, expVec.Cols, res)
}

func TestGetColumnNames(t *testing.T) {
	t.Parallel()
	t.Run("simple_select", func(t *testing.T) {
		t.Parallel()
		stmts, err := parser.Parse(`select id, name from clients`)
		require.NoError(t, err)
		require.Len(t, stmts, 1)

		cols, err := validator.GetColumnNames(stmts[0].AST)
		require.NoError(t, err)
		require.Equal(t, []string{"id", "name"}, cols)
	})

	t.Run("star_select_is_not_ok", func(t *testing.T) {
		t.Parallel()
		stmts, err := parser.Parse(`select * from clients`)
		require.NoError(t, err)
		require.Len(t, stmts, 1)

		_, err2 := validator.GetColumnNames(stmts[0].AST)
		require.ErrorIs(t, err2, validator.ErrAccessDenied)
	})
}
