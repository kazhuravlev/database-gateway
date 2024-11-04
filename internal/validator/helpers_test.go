package validator_test

import (
	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
	"testing"
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
	}

	for _, row := range table {
		t.Run(row.name, func(t *testing.T) {
			stmts, err := parser.Parse(row.sql)
			require.Nil(t, err)
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
