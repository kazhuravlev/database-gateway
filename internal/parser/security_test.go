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
