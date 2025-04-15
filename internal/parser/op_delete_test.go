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

func TestParseDeleteValid(t *testing.T) {
	t.Parallel()

	test := func(input string) {
		t.Helper()
		t.Run("", func(t *testing.T) {
			t.Helper()
			_, err := parser.Parse(input)
			require.NoError(t, err)
		})
	}

	test("DELETE FROM t1")
	test("DELETE FROM t1 WHERE c1 = 'value'")
	test("DELETE FROM t1 WHERE c1 = 123")
	test("DELETE FROM t1 WHERE c1 = NULL")
	test("DELETE FROM t1 WHERE c1 IS NULL")
	test("DELETE FROM t1 WHERE c1 IS NOT NULL")

	test("DELETE FROM s1.t1")
	test("DELETE FROM \"schema-with-dash\".\"table-with-dash\"")
	test("DELETE FROM s1.t1 WHERE c1 = 'value'")

	test("DELETE FROM t1 WHERE c1 > 10")
	test("DELETE FROM t1 WHERE c1 >= 10 AND c2 <= 20")
	test("DELETE FROM t1 WHERE c1 = 'value' OR c2 = 123")
	test("DELETE FROM t1 WHERE c1 IN (1, 2, 3)")
	test("DELETE FROM t1 WHERE c1 NOT IN (1, 2, 3)")
	test("DELETE FROM t1 WHERE c1 LIKE 'prefix%'")
	test("DELETE FROM t1 WHERE c1 BETWEEN 1 AND 10")
	test("DELETE FROM t1 WHERE c1 = 'value' AND (c2 > 10 OR c3 < 20)")

	test("DELETE FROM t1 RETURNING c1")
	test("DELETE FROM t1 WHERE c1 = 'value' RETURNING c1, c2")
}

func TestParseDeleteInvalid(t *testing.T) {
	t.Parallel()

	test := func(input, name string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()
			_, err := parser.Parse(input)
			require.Error(t, err)
		})
	}

	test("DELETE t1", "missing_from")
	test("DELETE FROM", "missing_table")
	test("DELETE FROM t1 WHERE", "incomplete_where")
	test("DELETE FROM t1 WHERE c1 = (select 1)", "incomplete_condition")
	test("DELETE FROM t1 RETURNING *", "returning_all")
	test("DELETE FROM t1 WHERE c1 IN (select 1)", "empty_in_list")
	test("DELETE FROM t1 WHERE c1 = (SELECT c1 FROM t2)", "subquery_in_where")
	test("DELETE FROM t1 t1, t2", "multiple_tables")
	test("DELETE FROM t1 USING t2", "using_clause")
	test("DELETE FROM t1 WHERE EXISTS (SELECT 1 FROM t2)", "exists_subquery")
}
