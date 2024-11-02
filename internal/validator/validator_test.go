package validator_test

import (
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	t.Run("bad_requests", func(t *testing.T) {
		t.Run("query_has_no_statements", func(t *testing.T) {
			query := ``
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})

		t.Run("not_a_query", func(t *testing.T) {
			query := `what time is it?`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})

		t.Run("query_has_several_statements", func(t *testing.T) {
			query := `select 1; select 1`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})

		t.Run("star_select", func(t *testing.T) {
			query := `select * from table`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})

		t.Run("schema_changes", func(t *testing.T) {
			query := `create table aaa(id text);`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})

		t.Run("alter_table", func(t *testing.T) {
			query := `alter table aaa add column id text default '';`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})
	})

	t.Run("some_cases", func(t *testing.T) {
		t.Run("complicated_query", func(t *testing.T) {
			target := config.Target{Id: "t1"}
			acls := []config.ACL{{
				Op:     config.OpSelect,
				Target: "t1",
				Tbl:    "clients",
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
			err := validator.IsAllowed(target, config.User{Acls: acls}, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})
	})

	t.Run("select", func(t *testing.T) {
		t.Run("simple_allowed", func(t *testing.T) {
			target := config.Target{Id: "t1"}
			acls := []config.ACL{{
				Op:     config.OpSelect,
				Target: "t1",
				Tbl:    "clients",
				Allow:  true,
			}}
			query := `select id, name from clients;`
			err := validator.IsAllowed(target, config.User{Acls: acls}, query)
			require.NoError(t, err)
		})
		t.Run("simple_denied", func(t *testing.T) {
			target := config.Target{Id: "t1"}
			acls := []config.ACL{{
				Op:     config.OpSelect,
				Target: "t1",
				Tbl:    "clients",
				Allow:  false,
			}}
			query := `select id, name from clients;`
			err := validator.IsAllowed(target, config.User{Acls: acls}, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})

		t.Run("select_from_allowed_select__is_not_allowed", func(t *testing.T) {
			// TODO: make it allowed.
			target := config.Target{Id: "t1"}
			acls := []config.ACL{{
				Op:     config.OpSelect,
				Target: "t1",
				Tbl:    "clients",
				Allow:  true,
			}}
			query := `select id, name from (select id, name from clients)`
			err := validator.IsAllowed(target, config.User{Acls: acls}, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})
	})

	t.Run("update", func(t *testing.T) {
		t.Run("simple_allowed", func(t *testing.T) {
			target := config.Target{Id: "t1"}
			acls := []config.ACL{{
				Op:     config.OpUpdate,
				Target: "t1",
				Tbl:    "clients",
				Allow:  true,
			}}
			query := `update clients set id=1 and name='john'`
			err := validator.IsAllowed(target, config.User{Acls: acls}, query)
			require.NoError(t, err)
		})
		t.Run("simple_denied", func(t *testing.T) {
			target := config.Target{Id: "t1"}
			acls := []config.ACL{{
				Op:     config.OpUpdate,
				Target: "t1",
				Tbl:    "clients",
				Allow:  false,
			}}
			query := `update clients set id=1 and name='john'`
			err := validator.IsAllowed(target, config.User{Acls: acls}, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})
	})

	t.Run("delete", func(t *testing.T) {
		t.Run("simple_allowed", func(t *testing.T) {
			target := config.Target{Id: "t1"}
			acls := []config.ACL{{
				Op:     config.OpDelete,
				Target: "t1",
				Tbl:    "clients",
				Allow:  true,
			}}
			query := `delete from clients where id=42`
			err := validator.IsAllowed(target, config.User{Acls: acls}, query)
			require.NoError(t, err)
		})
		t.Run("simple_denied", func(t *testing.T) {
			target := config.Target{Id: "t1"}
			acls := []config.ACL{{
				Op:     config.OpDelete,
				Target: "t1",
				Tbl:    "clients",
				Allow:  false,
			}}
			query := `delete from clients where id=42`
			err := validator.IsAllowed(target, config.User{Acls: acls}, query)
			require.ErrorIs(t, err, validator.ErrAccessDenied)
		})
	})
}

func TestVector(t *testing.T) {
	vec, err := validator.MakeSelectVec(&tree.Select{
		With: nil,
		Select: &tree.SelectClause{
			Distinct: false,
			DistinctOn: tree.DistinctOn{
				&tree.ColumnItem{
					ColumnName: "distinct_col",
				},
			},
			Exprs: tree.SelectExprs{
				{
					Expr: &tree.ColumnItem{
						ColumnName: "col_1",
					},
					As: "alias_1",
				},
				{
					Expr: &tree.FuncExpr{
						Func: tree.WrapFunction("sum"),
						Type: 0,
						Exprs: tree.Exprs{
							&tree.ColumnItem{
								ColumnName: "col_2",
							},
						},
						Filter:    nil,
						WindowDef: nil,
						OrderBy:   nil,
					},
					As: "alias_2",
				},
			},
			From: tree.From{
				Tables: tree.TableExprs{
					tree.NewTableName("", "clients"),
				},
				AsOf: tree.AsOfClause{},
			},
			Where: &tree.Where{
				Expr: &tree.ColumnItem{
					ColumnName: "where_1",
				},
			},
			GroupBy: tree.GroupBy{
				&tree.ColumnItem{
					ColumnName: "group_1",
				},
			},
			Having: &tree.Where{
				Expr: &tree.ColumnItem{
					ColumnName: "having_1",
				},
			},
			Window: tree.Window{
				{
					Name:    "",
					RefName: "",
					Partitions: tree.Exprs{
						&tree.ColumnItem{
							ColumnName: "part_1",
						},
					},
					OrderBy: tree.OrderBy{
						{
							Expr: &tree.ColumnItem{
								ColumnName: "part_order_col",
							},
						},
					},
					Frame: nil,
				},
			},
			TableSelect: false,
		},
		OrderBy: tree.OrderBy{
			{
				Expr: &tree.ColumnItem{
					ColumnName: "order_col",
				},
			},
		},
		Limit:   nil,
		Locking: nil,
	})
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
}
