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
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/parser"
	"github.com/stretchr/testify/require"
)

func TestParseSelectClassifiesClauseColumns(t *testing.T) {
	t.Parallel()

	vecs, err := parser.Parse(`
SELECT target_col
FROM clients
WHERE filter_col = 1
GROUP BY group_col
ORDER BY sort_col
`)
	require.NoError(t, err)
	require.Len(t, vecs, 1)

	sel, ok := vecs[0].(parser.SelectVec)
	require.True(t, ok)
	require.Equal(t, "clients", sel.Tbl)
	require.Equal(t, []string{"target_col"}, sel.Target)
	require.Equal(t, []string{"filter_col"}, sel.Filter)
	require.Equal(t, []string{"group_col"}, sel.Group)
	require.Equal(t, []string{"sort_col"}, sel.Sort)
	require.NotContains(t, sel.Target, "filter_col")
	require.NotContains(t, sel.Target, "group_col")
	require.NotContains(t, sel.Target, "sort_col")
}

func TestParseReturningColumnsAreNotClassifiedAsTargets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		query      string
		assertions func(t *testing.T, vec parser.Vector)
	}{
		{
			name:  "insert",
			query: `INSERT INTO clients(name) VALUES ('alice') RETURNING id`,
			assertions: func(t *testing.T, vec parser.Vector) {
				t.Helper()

				ins, ok := vec.(parser.InsertVec)
				require.True(t, ok)
				require.Equal(t, []string{"name"}, ins.Target)
				require.Equal(t, []string{"id"}, ins.Returning)
				require.NotContains(t, ins.Target, "id")
			},
		},
		{
			name:  "update",
			query: `UPDATE clients SET name = 'alice' WHERE email = 'a@example.com' RETURNING id`,
			assertions: func(t *testing.T, vec parser.Vector) {
				t.Helper()

				upd, ok := vec.(parser.UpdateVec)
				require.True(t, ok)
				require.Equal(t, []string{"name"}, upd.Target)
				require.Equal(t, []string{"email"}, upd.Filter)
				require.Equal(t, []string{"id"}, upd.Returning)
				require.NotContains(t, upd.Target, "email")
				require.NotContains(t, upd.Target, "id")
			},
		},
		{
			name:  "delete",
			query: `DELETE FROM clients WHERE email = 'a@example.com' RETURNING id`,
			assertions: func(t *testing.T, vec parser.Vector) {
				t.Helper()

				del, ok := vec.(parser.DeleteVec)
				require.True(t, ok)
				require.Empty(t, del.Target)
				require.Equal(t, []string{"email"}, del.Filter)
				require.Equal(t, []string{"id"}, del.Returning)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			vecs, err := parser.Parse(tc.query)
			require.NoError(t, err)
			require.Len(t, vecs, 1)

			tc.assertions(t, vecs[0])
		})
	}
}
