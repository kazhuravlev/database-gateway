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

// DELETE must still produce an operation vector even when no columns are referenced,
// otherwise ACL checks can be silently bypassed.
func TestParseDeleteWithoutColumnsStillEmitsVector(t *testing.T) {
	t.Parallel()

	vecs, err := parser.Parse("DELETE FROM clients")
	require.NoError(t, err)
	require.Len(t, vecs, 1)

	del, ok := vecs[0].(parser.DeleteVec)
	require.True(t, ok)
	require.Equal(t, "clients", del.Tbl)
}

// Nested expressions in filters must contribute referenced columns,
// otherwise protected columns can be used without appearing in ACL vectors.
func TestParseUpdateWhereNestedExprTracksColumns(t *testing.T) {
	t.Parallel()

	vecs, err := parser.Parse("UPDATE clients SET public_col = 1 WHERE (secret_col + 1) > 10")
	require.NoError(t, err)
	require.Len(t, vecs, 1)

	upd, ok := vecs[0].(parser.UpdateVec)
	require.True(t, ok)
	require.Contains(t, upd.Columns(), "secret_col")
}

// SELECT with aggregate-only target must still emit a vector for ACL checks.
func TestParseSelectAggregateOnlyStillEmitsVector(t *testing.T) {
	t.Parallel()

	vecs, err := parser.Parse("SELECT count(*) FROM clients")
	require.NoError(t, err)
	require.Len(t, vecs, 1)

	sel, ok := vecs[0].(parser.SelectVec)
	require.True(t, ok)
	require.Equal(t, "clients", sel.Tbl)
}

func TestParseSelectQualifiedAggregateStarStillEmitsVector(t *testing.T) {
	t.Parallel()

	vecs, err := parser.Parse("SELECT count(c.*) FROM clients AS c")
	require.NoError(t, err)
	require.Len(t, vecs, 1)

	sel, ok := vecs[0].(parser.SelectVec)
	require.True(t, ok)
	require.Equal(t, "clients", sel.Tbl)
}

// INSERT without explicit column list must still emit a vector for ACL checks.
func TestParseInsertImplicitColumnsStillEmitsVector(t *testing.T) {
	t.Parallel()

	vecs, err := parser.Parse("INSERT INTO clients VALUES (1)")
	require.NoError(t, err)
	require.Len(t, vecs, 1)

	ins, ok := vecs[0].(parser.InsertVec)
	require.True(t, ok)
	require.Equal(t, "clients", ins.Tbl)
}

// Boolean WHERE clauses must contribute every referenced column to vectors.
func TestParseSelectBoolWhereTracksAllColumns(t *testing.T) {
	t.Parallel()

	vecs, err := parser.Parse("SELECT id FROM clients WHERE id = 1 AND secret_col = 2")
	require.NoError(t, err)
	require.Len(t, vecs, 1)

	sel, ok := vecs[0].(parser.SelectVec)
	require.True(t, ok)
	require.Contains(t, sel.Columns(), "secret_col")
}

// Null tests must contribute referenced columns,
// otherwise protected columns can be used in filters without appearing in ACL vectors.
func TestParseSelectNullTestTracksColumn(t *testing.T) {
	t.Parallel()

	vecs, err := parser.Parse("SELECT id FROM clients WHERE secret_col IS NOT NULL")
	require.NoError(t, err)
	require.Len(t, vecs, 1)

	sel, ok := vecs[0].(parser.SelectVec)
	require.True(t, ok)
	require.Contains(t, sel.Columns(), "secret_col")
}

// Unsupported WHERE nodes should be rejected, not silently ignored.
func TestParseSelectUnsupportedWhereExprReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parser.Parse("SELECT id FROM clients WHERE EXISTS (SELECT 1)")
	require.Error(t, err)
}
