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

// Unsupported WHERE nodes should be rejected, not silently ignored.
func TestParseSelectUnsupportedWhereExprReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parser.Parse("SELECT id FROM clients WHERE EXISTS (SELECT 1)")
	require.Error(t, err)
}
