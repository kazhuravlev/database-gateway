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

func TestParseColumn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tokens      []string
		wantTable   string
		wantColumn  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "column only",
			tokens:     []string{"col"},
			wantTable:  "",
			wantColumn: "col",
		},
		{
			name:       "table and column",
			tokens:     []string{"tbl", "col"},
			wantTable:  "tbl",
			wantColumn: "col",
		},
		{
			name:       "schema table and column",
			tokens:     []string{"schema", "tbl", "col"},
			wantTable:  "schema.tbl",
			wantColumn: "col",
		},
		{
			name:       "catalog schema table and column",
			tokens:     []string{"catalog", "schema", "tbl", "col"},
			wantTable:  "catalog.schema.tbl",
			wantColumn: "col",
		},
		{
			name:        "zero tokens",
			tokens:      nil,
			wantErr:     true,
			errContains: "expected 1-4 tokens",
		},
		{
			name:        "five tokens exceeds max",
			tokens:      []string{"a", "b", "c", "d", "e"},
			wantErr:     true,
			errContains: "expected 1-4 tokens",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			col, err := parser.ParseColumn(tc.tokens...)
			if tc.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errContains)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantTable, col.Table())
			require.Equal(t, []string{tc.wantColumn}, parser.Columns{col}.ListNames())
		})
	}
}
