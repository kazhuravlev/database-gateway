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

package parser_test //nolint:exhaustruct

import (
	"sort"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/parser"
	"github.com/stretchr/testify/require"
)

// TestSupportedQueries is a regression net that pins down the set of queries the
// parser supports right now, together with the tables and columns it extracts.
// Every case here parses without an error in the current implementation; if a
// future change to the parser or validation breaks one of these constructions,
// the corresponding case must fail loudly instead of silently narrowing support.
func TestSupportedQueries(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		query string
		want  []parser.Vector
	}{
		// Simple SELECT with a column list.
		{
			name:  "select_column_list",
			query: `SELECT id, name FROM clients`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"id", "name"}, Filter: []string{}, Group: []string{}, Sort: []string{}},
			},
		},
		// Qualified and unqualified table names.
		{
			name:  "select_schema_qualified_table",
			query: `SELECT id FROM public.clients`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "public.clients", Target: []string{"id"}, Filter: []string{}, Group: []string{}, Sort: []string{}},
			},
		},
		{
			name:  "select_table_qualified_column",
			query: `SELECT clients.id FROM clients`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"id"}, Filter: []string{}, Group: []string{}, Sort: []string{}},
			},
		},
		{
			name:  "select_fully_qualified_column",
			query: `SELECT public.clients.id FROM public.clients`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "public.clients", Target: []string{"id"}, Filter: []string{}, Group: []string{}, Sort: []string{}},
			},
		},
		{
			name:  "select_quoted_identifiers",
			query: `SELECT "id" FROM "public"."clients"`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "public.clients", Target: []string{"id"}, Filter: []string{}, Group: []string{}, Sort: []string{}},
			},
		},
		// Table aliases.
		{
			name:  "select_table_alias",
			query: `SELECT c.id FROM clients AS c`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"id"}, Filter: []string{}, Group: []string{}, Sort: []string{}},
			},
		},
		// WHERE.
		{
			name:  "select_where_comparisons",
			query: `SELECT id FROM clients WHERE status = 'active' AND age > 18`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"id"}, Filter: []string{"status", "age"}, Group: []string{}, Sort: []string{}},
			},
		},
		{
			name:  "select_where_is_not_null",
			query: `SELECT id FROM clients WHERE deleted_at IS NOT NULL`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"id"}, Filter: []string{"deleted_at"}, Group: []string{}, Sort: []string{}},
			},
		},
		// ORDER BY.
		{
			name:  "select_order_by",
			query: `SELECT id FROM clients ORDER BY name DESC`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"id"}, Filter: []string{}, Group: []string{}, Sort: []string{"name"}},
			},
		},
		// LIMIT / OFFSET.
		{
			name:  "select_limit_offset",
			query: `SELECT id FROM clients LIMIT 10 OFFSET 5`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"id"}, Filter: []string{}, Group: []string{}, Sort: []string{}},
			},
		},
		{
			name:  "select_limit_all",
			query: `SELECT id FROM clients LIMIT ALL`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"id"}, Filter: []string{}, Group: []string{}, Sort: []string{}},
			},
		},
		// GROUP BY.
		{
			name:  "select_group_by",
			query: `SELECT c1, c2 FROM clients GROUP BY c1, c2`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"c1", "c2"}, Filter: []string{}, Group: []string{"c1", "c2"}, Sort: []string{}},
			},
		},
		// Allowed functions. Note: count(*) yields no target columns.
		{
			name:  "select_count_star",
			query: `SELECT count(*) FROM clients`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{}, Filter: []string{}, Group: []string{}, Sort: []string{}},
			},
		},
		{
			name:  "select_function_with_column_arg",
			query: `SELECT lower(name) FROM clients`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"name"}, Filter: []string{}, Group: []string{}, Sort: []string{}},
			},
		},
		// JOIN with ON condition; the ON columns land in Filter of each table.
		{
			name:  "select_join_on",
			query: `SELECT clients.id, orders.total FROM clients JOIN orders ON clients.id = orders.client_id`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"id"}, Filter: []string{"id"}, Group: []string{}, Sort: []string{}},
				parser.SelectVec{Tbl: "orders", Target: []string{"total"}, Filter: []string{"client_id"}, Group: []string{}, Sort: []string{}},
			},
		},
		{
			name:  "select_left_join_on",
			query: `SELECT clients.id FROM clients LEFT JOIN orders ON clients.id = orders.client_id`,
			want: []parser.Vector{
				parser.SelectVec{Tbl: "clients", Target: []string{"id"}, Filter: []string{"id"}, Group: []string{}, Sort: []string{}},
				parser.SelectVec{Tbl: "orders", Target: []string{}, Filter: []string{"client_id"}, Group: []string{}, Sort: []string{}},
			},
		},
		// INSERT.
		{
			name:  "insert_values",
			query: `INSERT INTO clients (id, name) VALUES (1, 'bob')`,
			want: []parser.Vector{
				parser.InsertVec{Tbl: "clients", Target: []string{"id", "name"}, Returning: []string{}},
			},
		},
		{
			name:  "insert_returning",
			query: `INSERT INTO public.clients (id) VALUES (1) RETURNING id, name`,
			want: []parser.Vector{
				parser.InsertVec{Tbl: "public.clients", Target: []string{"id"}, Returning: []string{"id", "name"}},
			},
		},
		{
			name:  "insert_on_conflict_do_nothing",
			query: `INSERT INTO clients (id) VALUES (1) ON CONFLICT (id) DO NOTHING`,
			want: []parser.Vector{
				parser.InsertVec{Tbl: "clients", Target: []string{"id", "id"}, Returning: []string{}},
			},
		},
		// UPDATE.
		{
			name:  "update_where",
			query: `UPDATE clients SET name = 'bob' WHERE id = 1`,
			want: []parser.Vector{
				parser.UpdateVec{Tbl: "clients", Target: []string{"name"}, Filter: []string{"id"}, Returning: []string{}},
			},
		},
		{
			name:  "update_qualified_returning",
			query: `UPDATE public.clients SET name = 'bob', age = 2 WHERE id = 1 RETURNING id`,
			want: []parser.Vector{
				parser.UpdateVec{Tbl: "public.clients", Target: []string{"name", "age"}, Filter: []string{"id"}, Returning: []string{"id"}},
			},
		},
		// DELETE.
		{
			name:  "delete_where",
			query: `DELETE FROM clients WHERE id = 1`,
			want: []parser.Vector{
				parser.DeleteVec{Tbl: "clients", Target: nil, Filter: []string{"id"}, Returning: []string{}},
			},
		},
		{
			name:  "delete_qualified_returning",
			query: `DELETE FROM public.clients WHERE id = 1 RETURNING id, name`,
			want: []parser.Vector{
				parser.DeleteVec{Tbl: "public.clients", Target: nil, Filter: []string{"id"}, Returning: []string{"id", "name"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parser.Parse(tc.query)
			require.NoError(t, err)

			// Vector order is not guaranteed for multi-table queries.
			sortVectors(got)
			sortVectors(tc.want)

			require.Equal(t, tc.want, got)
		})
	}
}

func sortVectors(vecs []parser.Vector) {
	sort.Slice(vecs, func(i, j int) bool {
		return vectorTable(vecs[i]) < vectorTable(vecs[j])
	})
}

func vectorTable(v parser.Vector) string {
	switch vec := v.(type) {
	case parser.SelectVec:
		return vec.Tbl
	case parser.InsertVec:
		return vec.Tbl
	case parser.UpdateVec:
		return vec.Tbl
	case parser.DeleteVec:
		return vec.Tbl
	default:
		return ""
	}
}
