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

func TestParseUpdateValid(t *testing.T) {
	t.Parallel()

	test := func(input string) {
		t.Helper()
		t.Run("", func(t *testing.T) {
			t.Helper()
			_, err := parser.Parse(input)
			require.NoError(t, err)
		})
	}

	test("UPDATE t1 SET c1 = 'value'")
	test("UPDATE t1 SET c1 = 'value', c2 = 123")
	test("UPDATE t1 SET c1 = NULL")
	test("UPDATE t1 SET c1 = DEFAULT")

	test("UPDATE s1.t1 SET c1 = 'value'")
	test("UPDATE \"schema-with-dash\".\"table-with-dash\" SET c1 = 'value'")

	test("UPDATE t1 SET c1 = 'value' WHERE c2 = 1")
	test("UPDATE t1 SET c1 = 'value' WHERE c2 > 10 AND c3 = true")
	test("UPDATE t1 SET c1 = 'value' WHERE c2 IN (1, 2, 3)")
	test("UPDATE t1 SET c1 = 'value' WHERE c2 LIKE 'prefix%'")
	test("UPDATE t1 SET c1 = 'value' WHERE c2 IS NULL")

	test("UPDATE t1 SET c1 = c1 + 1")
	test("UPDATE t1 SET c1 = c1 * 1.1")
	test("UPDATE t1 SET c1 = c1 || ' ' || c2")

	test("UPDATE t1 SET c1 = 'value' RETURNING c2")
	test("UPDATE t1 SET c1 = 'value' RETURNING c2, c1")
}

func TestParseUpdateInvalid(t *testing.T) {
	t.Parallel()

	test := func(input, name string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()
			_, err := parser.Parse(input)
			require.Error(t, err)
		})
	}

	test("UPDATE t1 SET c1 = 'v1' RETURNING *", "start_returning")
	test("UPDATE t1 SET c1 = t2.c1 FROM t2 WHERE t1.id = t2.id", "update_from")
	test("UPDATE t1 SET c1 = t2.c1 FROM t2 WHERE t1.id = t2.id", "update_from")
	test("UPDATE t1 SET c1 = (SELECT MAX(c1) FROM t2)", "subquery_in_target_field")
	test("UPDATE t1 SET c1 = 'value' WHERE id IN (SELECT id FROM t2 WHERE active = true)", "subquery_in_where")
}
