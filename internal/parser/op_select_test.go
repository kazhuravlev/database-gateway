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

package parser_test

import (
	"sort"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/parser"
	"github.com/stretchr/testify/require"
)

func TestParseSelectValid(t *testing.T) {
	t.Parallel()

	fValid := func(input string) {
		t.Helper()
		t.Run(input, func(t *testing.T) {
			t.Helper()
			_, err := parser.Parse(input)
			require.NoError(t, err)
		})
	}

	fValid("SELECT public.clients.id from public.clients")
	fValid("SELECT clients.id from public.clients")
	fValid("SELECT id from public.clients")

	fValid("SELECT public.clients.id from clients")
	fValid("SELECT clients.id from clients")
	fValid("SELECT id from clients")

	fValid("SELECT id, id from clients")
	fValid(`SELECT "id" from "clients"`)

	fValid(`SELECT public.clients.c1, clients.c2, c3 from "clients"`)
	fValid(`SELECT clients.id AS client_id from clients`)
	fValid(`SELECT "public"."clients"."id" from "public"."clients"`)

	fValid(`SELECT id FROM clients WHERE id > 0`)
	fValid(`SELECT id FROM clients WHERE clients.id > 0`)
	fValid(`SELECT id FROM clients WHERE id = 42`)
	fValid(`SELECT id FROM clients WHERE id != 42`)
	fValid(`SELECT id FROM clients WHERE id > 42`)
	fValid(`SELECT id FROM clients WHERE id < 42`)
	fValid(`SELECT id FROM clients WHERE id is null`)
	fValid(`SELECT id FROM clients WHERE id is NULL`)
	fValid(`SELECT id FROM clients WHERE id is not NULL`)

	fValid(`SELECT id FROM clients ORDER BY id ASC`)
	fValid(`SELECT id FROM clients ORDER BY id DESC`)
	fValid(`SELECT id FROM clients LIMIT 10`)
	fValid(`SELECT id FROM clients OFFSET 5`)
	fValid(`SELECT id FROM clients LIMIT 10 OFFSET 5`)

	fValid(`SELECT c1, c2 FROM clients GROUP BY c1, c2`)
	fValid(`SELECT c1 FROM clients GROUP BY c1`)
	fValid(`SELECT c1 FROM clients GROUP BY clients.c1`)
	fValid(`SELECT c1 FROM clients GROUP BY public.clients.c1`)
	fValid(`SELECT c1 FROM clients GROUP BY "public"."clients"."c1"`)

	fValid(`SELECT c1 FROM "schema-with-dash"."table-with-dash"`)
	fValid(`SELECT c1 FROM schema1.clients`)

	fValid(`SELECT lower(id) from clients`)
	fValid(`SELECT count(*) from clients`)
	fValid(`SELECT sum(total_sales) from sales_olap_42`)
}

// TestParseSelectInvalid is just a list of special cases that is not supported (yet).
func TestParseSelectInvalid(t *testing.T) {
	t.Parallel()

	fInvalid := func(input, name string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()
			_, err := parser.Parse(input)
			require.Error(t, err)
		})
	}

	fInvalid("SELECT 1", "select_without_from")
	fInvalid("SELECT * FROM clients", "star_expression")         //nolint:unqueryvet
	fInvalid("SELECT clients.* FROM clients", "star_expression") //nolint:unqueryvet
	fInvalid("SELECT public.clients.* FROM clients", "star_expression")

	fInvalid("SELECT * FROM public.clients", "star_expression_scm")         //nolint:unqueryvet
	fInvalid("SELECT clients.* FROM public.clients", "star_expression_scm") //nolint:unqueryvet
	fInvalid("SELECT public.clients.* FROM public.clients", "star_expression_scm")

	fInvalid("SELECT id FROM (SELECT id FROM clients) AS sub", "nested_select")
	fInvalid("SELECT id FROM clients UNION SELECT id FROM orders", "union_expression")
}

func TestParseSelectJoinVectors(t *testing.T) {
	t.Parallel()

	type expectedVec struct {
		tbl  string
		cols []string
	}

	testCases := []struct {
		name  string
		query string
		exp   []expectedVec
	}{
		{
			name:  "inner_join_with_explicit_columns",
			query: `SELECT clients.id, orders.id FROM clients INNER JOIN orders ON clients.id = orders.client_id`,
			exp: []expectedVec{
				{tbl: "clients", cols: []string{"id"}},
				{tbl: "orders", cols: []string{"client_id", "id"}},
			},
		},
		{
			name: "left_join_with_aliases_and_predicates",
			query: `SELECT c.id, count(o.id)
FROM clients AS c
LEFT JOIN orders AS o ON c.id = o.client_id
WHERE o.status = 'paid'
GROUP BY c.id
ORDER BY c.id`,
			exp: []expectedVec{
				{tbl: "clients", cols: []string{"id"}},
				{tbl: "orders", cols: []string{"client_id", "id", "status"}},
			},
		},
		{
			name: "schema_qualified_join",
			query: `SELECT public.clients.id, billing.orders.amount
FROM public.clients
JOIN billing.orders ON public.clients.id = billing.orders.client_id
WHERE billing.orders.status = 'paid'`,
			exp: []expectedVec{
				{tbl: "billing.orders", cols: []string{"amount", "client_id", "status"}},
				{tbl: "public.clients", cols: []string{"id"}},
			},
		},
		{
			name: "self_join_with_qualified_columns",
			query: `SELECT c.id, parent.name
FROM clients AS c
JOIN clients AS parent ON c.parent_id = parent.id
WHERE parent.active = true`,
			exp: []expectedVec{
				{tbl: "clients", cols: []string{"active", "id", "name", "parent_id"}},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			vecs, err := parser.Parse(tc.query)
			require.NoError(t, err)
			require.Len(t, vecs, len(tc.exp))

			got := make([]expectedVec, 0, len(vecs))
			for _, vec := range vecs {
				sel, ok := vec.(parser.SelectVec)
				require.True(t, ok)

				cols := sel.Columns()
				sort.Strings(cols)
				got = append(got, expectedVec{
					tbl:  sel.Tbl,
					cols: cols,
				})
			}

			sort.Slice(got, func(i, j int) bool {
				return got[i].tbl < got[j].tbl
			})

			require.Equal(t, tc.exp, got)
		})
	}
}

func TestParseSelectJoinInvalid(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		query string
	}{
		{
			name:  "join_using_clause",
			query: `SELECT c.id FROM clients AS c JOIN orders AS o USING (id)`,
		},
		{
			name:  "natural_join",
			query: `SELECT c.id FROM clients AS c NATURAL JOIN orders AS o`,
		},
		{
			name:  "join_subquery_rhs",
			query: `SELECT c.id FROM clients AS c JOIN (SELECT client_id FROM orders) AS o ON c.id = o.client_id`,
		},
		{
			name:  "ambiguous_unqualified_column_in_join_predicate",
			query: `SELECT c.id FROM clients AS c JOIN orders AS o ON id = o.client_id`,
		},
		{
			name:  "ambiguous_unqualified_column_in_self_join_target",
			query: `SELECT id FROM clients AS c JOIN clients AS parent ON c.parent_id = parent.id`,
		},
		{
			name:  "ambiguous_unqualified_column_in_self_join_filter",
			query: `SELECT c.id FROM clients AS c JOIN clients AS parent ON c.parent_id = parent.id WHERE name IS NOT NULL`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := parser.Parse(tc.query)
			require.Error(t, err)
		})
	}
}
