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
	"strings"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/parser"
	"github.com/stretchr/testify/require"
)

func TestRejectUnaccountedSQL(t *testing.T) {
	t.Parallel()
	for name, query := range map[string]string{
		"returning_subquery_alias":   "DELETE FROM clients RETURNING (SELECT secret FROM secrets) AS id",
		"returning_function_alias":   "UPDATE clients SET id=1 RETURNING pg_read_file('/etc/passwd') AS id",
		"returning_arithmetic_alias": "INSERT INTO clients(id) VALUES(1) RETURNING secret+1 AS id",
		"upsert_subquery":            "INSERT INTO clients(id) VALUES(1) ON CONFLICT(id) DO UPDATE SET id=(SELECT secret FROM secrets)",
		"upsert_function":            "INSERT INTO clients(id) VALUES(1) ON CONFLICT(id) DO UPDATE SET id=evil()",
		"column_alias_list":          "SELECT id FROM clients AS c(id,other_id)",
		"quoted_table_dot":           `SELECT id FROM "public.clients"`,
		"quoted_schema_dot":          `SELECT id FROM "public.clients".secrets`,
		"quoted_column_dot":          `SELECT "c.id" FROM clients c`,
		"cross_database":             "SELECT id FROM other.public.clients",
		"duplicate_alias":            "SELECT a.id FROM clients a JOIN secrets a ON a.id=a.id",
		"hidden_base_name":           "SELECT clients.id FROM clients c",
		"whole_row_function":         "SELECT lower(c.*) FROM clients c",
		"count_whole_row":            "SELECT count(c.*) FROM clients c",
		"quoted_function_case":       `SELECT "LOWER"(name) FROM clients`,
		"function_variadic":          "SELECT lower(VARIADIC names) FROM clients",
		"function_no_args":           "SELECT lower() FROM clients",
		"function_wrong_star":        "SELECT lower(*) FROM clients",
		"implicit_insert_columns":    "INSERT INTO clients VALUES(1)",
		"default_insert_columns":     "INSERT INTO clients DEFAULT VALUES",
		"column_indirection":         "UPDATE clients SET id[1]=2",
		"qualified_operator":         "SELECT id FROM clients WHERE id OPERATOR(public.=) 1",
		"cte_mutation":               "WITH x AS (DELETE FROM secrets RETURNING id) SELECT id FROM clients",
		"select_into":                "SELECT id INTO stolen FROM clients",
		"locking":                    "SELECT id FROM clients FOR UPDATE",
		"table_sampling":             "SELECT id FROM clients TABLESAMPLE SYSTEM(10)",
		"multiple_statements":        "SELECT id FROM clients; DELETE FROM secrets",
		"nul_truncation":             "SELECT id FROM clients\x00; DELETE FROM secrets",
		"invalid_utf8":               "SELECT id FROM clients -- \xff",
		"oversized":                  "SELECT id FROM clients -- " + strings.Repeat("x", parser.MaxQueryBytes),
		"deep":                       "SELECT id FROM clients WHERE " + strings.Repeat("(", 65) + "id=1" + strings.Repeat(")", 65),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			vectors, err := parser.Parse(query)
			require.Error(t, err)
			require.Nil(t, vectors)
		})
	}
}

func TestReturningAliasTracksSourceColumn(t *testing.T) {
	t.Parallel()
	for _, query := range []string{
		"DELETE FROM clients RETURNING secret AS id",
		"UPDATE clients SET id=1 RETURNING secret AS id",
		"INSERT INTO clients(id) VALUES(1) RETURNING secret AS id",
	} {
		vectors, err := parser.Parse(query)
		require.NoError(t, err)
		switch vector := vectors[0].(type) {
		case parser.DeleteVec:
			require.Equal(t, []string{"secret"}, vector.Returning)
		case parser.UpdateVec:
			require.Equal(t, []string{"secret"}, vector.Returning)
		case parser.InsertVec:
			require.Equal(t, []string{"secret"}, vector.Returning)
		default:
			t.Fatalf("unexpected vector %T", vector)
		}
	}
}

func TestAliasesCannotHideRelationsOrCycle(t *testing.T) {
	t.Parallel()
	for _, query := range []string{
		"SELECT a.id FROM clients a JOIN secrets clients ON a.id=clients.id",
		"SELECT secrets.id FROM clients secrets JOIN secrets clients ON secrets.id=clients.id",
	} {
		vectors, err := parser.Parse(query)
		require.NoError(t, err)
		require.Len(t, vectors, 2)
		require.Equal(t, "clients", vectorTable(vectors[0]))
		require.Equal(t, "secrets", vectorTable(vectors[1]))
	}
}

func TestCompileBindsNamesAndPreservesDependencies(t *testing.T) {
	t.Parallel()
	query := "SELECT lower(c.name) FROM clients c WHERE c.id=1"
	sql, vectors, err := parser.Compile(query, "tenant")
	require.NoError(t, err)
	require.Contains(t, sql, "tenant.clients")
	require.Contains(t, sql, "pg_catalog.lower")
	require.Equal(t, "tenant.clients", vectorTable(vectors[0]))
	again, err := parser.Parse(sql)
	require.NoError(t, err)
	require.Equal(t, vectors, again)
	sql2, vectors2, err := parser.Compile(sql, "tenant")
	require.NoError(t, err)
	require.Equal(t, sql, sql2)
	require.Equal(t, vectors, vectors2)
}

func FuzzParseAndCompile(f *testing.F) {
	for _, query := range []string{
		"SELECT id FROM clients", "SELECT count(*) FROM clients", "SELECT c.id FROM clients c JOIN orders o ON c.id=o.id",
		"INSERT INTO clients(id) VALUES(1) ON CONFLICT(id) DO UPDATE SET id=2 RETURNING id",
		"UPDATE clients SET id=id+1 WHERE id=1 RETURNING id", "DELETE FROM clients RETURNING secret AS id",
		"SELECT id FROM clients\x00;DELETE FROM secrets", "SELECT id FROM clients AS c(id,other_id)",
	} {
		f.Add(query)
	}
	f.Fuzz(func(t *testing.T, query string) {
		first, err := parser.Parse(query)
		if err != nil {
			require.Nil(t, first)

			return
		}
		require.NotEmpty(t, first)
		second, err := parser.Parse(query)
		require.NoError(t, err)
		require.Equal(t, first, second)
		sql, vectors, err := parser.Compile(query, "public")
		if err != nil {
			require.Empty(t, sql)
			require.Nil(t, vectors)

			return
		}
		parsed, err := parser.Parse(sql)
		require.NoError(t, err)
		require.Equal(t, vectors, parsed)
		sql2, again, err := parser.Compile(sql, "public")
		require.NoError(t, err)
		require.Equal(t, sql, sql2)
		require.Equal(t, vectors, again)
	})
}
