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
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

func TestGetColumnNames(t *testing.T) {
	t.Parallel()

	table := []struct {
		name    string
		sql     string
		expCols []string
		expErr  error
	}{
		{
			name:    `simple_select`,
			sql:     `select id, name from clients`,
			expCols: []string{"id", "name"},
			expErr:  nil,
		},
		{
			name:    `start_select_not_supported`,
			sql:     `select * from clients`,
			expCols: nil,
			expErr:  validator.ErrComplicatedQuery,
		},
		{
			name:    `select_count_star`,
			sql:     `select 1, count(*) from clients`,
			expCols: nil,
			expErr:  validator.ErrComplicatedQuery,
		},
		{
			name:    `select_count_column`,
			sql:     `select now(), 1, count(id) from clients`,
			expCols: []string{"id"},
			expErr:  nil,
		},
		{
			name:    `select_star_with_another_fields`,
			sql:     `select * from clients where id=1`,
			expCols: nil,
			expErr:  validator.ErrComplicatedQuery,
		},
		{
			name:    `select_star_with_another_fields_and_tbl_mention`,
			sql:     `select clients.* from clients where id=1`,
			expCols: nil,
			expErr:  validator.ErrComplicatedQuery,
		},
		{
			name:    `update`,
			sql:     `update clients set id=1`,
			expCols: []string{"id"},
			expErr:  nil,
		},
		{
			name:    `update_with_repeated_columns`,
			sql:     `update clients set id=1 where id=2`,
			expCols: []string{"id"},
			expErr:  nil,
		},
		{
			name:    `update_when_filter_inversed`,
			sql:     `update clients set name=1 where 2=id`,
			expCols: []string{"id", "name"},
			expErr:  nil,
		},
	}

	for _, row := range table {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()

			stmts, err := parser.Parse(row.sql)
			require.NoError(t, err)
			require.Len(t, stmts, 1)

			cols, err := validator.GetColumnNames(stmts[0].AST)
			if row.expErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, row.expErr)
				require.Nil(t, cols)
			} else {
				require.NoError(t, err)
				require.Len(t, cols, len(row.expCols))
				require.Equal(t, row.expCols, cols)
			}
		})
	}
}
