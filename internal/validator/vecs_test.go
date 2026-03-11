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
	"sort"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

func TestMakeVectors(t *testing.T) {
	t.Parallel()

	t.Run("happy_path", func(t *testing.T) {
		t.Parallel()
		test := func(name, query string, exp []validator.Vec) {
			t.Helper()
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				t.Helper()
				vecs, err := validator.MakeVectors(query)
				require.NoError(t, err)
				require.Len(t, vecs, len(exp))
				for i := range exp {
					sort.Strings(vecs[i].Cols)
					require.Equal(t, exp[i], vecs[i])
				}
			})
		}

		test("select_complex",
			`select f1, count(f2) from clients where f3=1 group by f4 order by f5`,
			[]validator.Vec{{
				Op:   config.OpSelect,
				Tbl:  "clients",
				Cols: []string{"f1", "f2", "f3", "f4", "f5"},
			}})
		test("insert_complex",
			`insert into clients (f1, f2) values (1, 2) on conflict (f3, f4) do update set f5=33 returning f6`,
			[]validator.Vec{{
				Op:   config.OpInsert,
				Tbl:  "clients",
				Cols: []string{"f1", "f2", "f3", "f4", "f5", "f6"},
			}})
		test("update_complex",
			`update clients set f1=1 where f2=2 returning f3`,
			[]validator.Vec{{
				Op:   config.OpUpdate,
				Tbl:  "clients",
				Cols: []string{"f1", "f2", "f3"},
			}})
		test("delete_complex",
			`delete from clients where f1=1 returning f2`,
			[]validator.Vec{{
				Op:   config.OpDelete,
				Tbl:  "clients",
				Cols: []string{"f1", "f2"},
			}})
	})

	t.Run("join_queries", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name  string
			query string
			exp   []validator.Vec
		}{
			{
				name:  "inner_join",
				query: `select clients.id, orders.id from clients inner join orders on clients.id = orders.client_id`,
				exp: []validator.Vec{
					{
						Op:   config.OpSelect,
						Tbl:  "clients",
						Cols: []string{"id"},
					},
					{
						Op:   config.OpSelect,
						Tbl:  "orders",
						Cols: []string{"client_id", "id"},
					},
				},
			},
			{
				name: "left_join_with_aliases",
				query: `select c.id, count(o.id)
from clients as c
left join orders as o on c.id = o.client_id
where o.status = 'paid'
group by c.id
order by c.id`,
				exp: []validator.Vec{
					{
						Op:   config.OpSelect,
						Tbl:  "clients",
						Cols: []string{"id"},
					},
					{
						Op:   config.OpSelect,
						Tbl:  "orders",
						Cols: []string{"client_id", "id", "status"},
					},
				},
			},
			{
				name: "schema_qualified_join",
				query: `select public.clients.id, billing.orders.amount
from public.clients
join billing.orders on public.clients.id = billing.orders.client_id
where billing.orders.status = 'paid'`,
				exp: []validator.Vec{
					{
						Op:   config.OpSelect,
						Tbl:  "billing.orders",
						Cols: []string{"amount", "client_id", "status"},
					},
					{
						Op:   config.OpSelect,
						Tbl:  "public.clients",
						Cols: []string{"id"},
					},
				},
			},
			{
				name: "self_join_with_qualified_columns",
				query: `select c.id, parent.name
from clients as c
join clients as parent on c.parent_id = parent.id
where parent.active = true`,
				exp: []validator.Vec{
					{
						Op:   config.OpSelect,
						Tbl:  "clients",
						Cols: []string{"active", "id", "name", "parent_id"},
					},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				vecs, err := validator.MakeVectors(tc.query)
				require.NoError(t, err)
				require.Len(t, vecs, len(tc.exp))

				for i := range vecs {
					sort.Strings(vecs[i].Cols)
				}

				sort.Slice(vecs, func(i, j int) bool {
					return vecs[i].Tbl < vecs[j].Tbl
				})

				require.Equal(t, tc.exp, vecs)
			})
		}
	})

	t.Run("bad_path", func(t *testing.T) {
		t.Parallel()
		test := func(name, query string, err error) {
			t.Helper()
			t.Run(name, func(t *testing.T) {
				t.Helper()
				t.Parallel()
				vecs, err2 := validator.MakeVectors(query)
				require.Error(t, err2)
				require.ErrorIs(t, err2, err)
				require.Nil(t, vecs)
			})
		}

		test("select_star",
			`select * from clients`, //nolint:unqueryvet
			validator.ErrComplicatedQuery)
		test("insert_star",
			`insert into clients(f1, f3) values(1,2) returning *`,
			validator.ErrComplicatedQuery)
		test("update_star",
			`update clients set f1=1 where f2=2 returning *`,
			validator.ErrComplicatedQuery)
		test("delete_star",
			`delete from clients where f1=1 returning *`,
			validator.ErrComplicatedQuery)
		test("join_using",
			`select c.id from clients as c join orders as o using (id)`,
			validator.ErrComplicatedQuery)
		test("natural_join",
			`select c.id from clients as c natural join orders as o`,
			validator.ErrComplicatedQuery)
		test("join_subquery_rhs",
			`select c.id from clients as c join (select client_id from orders) as o on c.id = o.client_id`,
			validator.ErrComplicatedQuery)
		test("join_with_ambiguous_column",
			`select c.id from clients as c join orders as o on id = o.client_id`,
			validator.ErrComplicatedQuery)
		test("self_join_with_ambiguous_target",
			`select id from clients as c join clients as parent on c.parent_id = parent.id`,
			validator.ErrComplicatedQuery)
		test("self_join_with_ambiguous_filter",
			`select c.id from clients as c join clients as parent on c.parent_id = parent.id where name is not null`,
			validator.ErrComplicatedQuery)
	})
}
