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

func TestParseInsertValid(t *testing.T) {
	t.Parallel()

	test := func(input string) {
		t.Helper()
		t.Run("", func(t *testing.T) {
			t.Helper()
			_, err := parser.Parse(input)
			require.NoError(t, err)
		})
	}

	test("insert into d1.s1.t1(c1) values(DEFAULT)")
	test("insert into s1.t1(c1) values(DEFAULT)")
	test("insert into t1(c1) values(DEFAULT)")

	test("insert into t1(c1) values(DEFAULT)")

	test("insert into t1(c1,c2,c3) values(1,'2',NULL)")

	test("INSERT INTO t1 VALUES (1, 'text')")
	test("INSERT INTO t1(c1, c2) VALUES (1, 'text')")
	test("INSERT INTO t1(c1, c2) VALUES (1, 'text with ''quotes''')")

	test("INSERT INTO s1.t1(id) VALUES (1)")
	test("INSERT INTO s1.t1(id, name) VALUES (1, 'text')")
	test("INSERT INTO \"schema-with-dash\".\"table-with-dash\"(id) VALUES (1)")

	test("INSERT INTO t1(id, name) VALUES (1, 'a'), (2, 'b'), (3, 'c')")
	test("INSERT INTO t1(id, name) VALUES (1, NULL), (2, DEFAULT)")

	test("INSERT INTO t1(id, name, created, active, data) VALUES (1, 'name', '2023-01-01', true, '{\"key\": \"value\"}')")
	test("INSERT INTO t1(id, salary) VALUES (1, 1000.50)")

	test("INSERT INTO t1(name) VALUES ('test') RETURNING id")
	test("INSERT INTO t1(name) VALUES ('test') RETURNING id, name")

	test("INSERT INTO t1(id, name) VALUES (1, 'test') ON CONFLICT (id) DO NOTHING")
	test("INSERT INTO t1(id, name) VALUES (1, 'test') ON CONFLICT (id) DO UPDATE SET name = 'updated'")

	test("INSERT INTO t1 DEFAULT VALUES")
}

func TestParseInsertInvalid(t *testing.T) {
	t.Parallel()

	test := func(input, name string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()
			_, err := parser.Parse(input)
			require.Error(t, err)
		})
	}

	test("insert into t1(c1) values(1) returning *", "star_returning")
	test("insert into t1(c1) select t2.c1 from t2", "insert_from_select")

	test("INSERT INTO t1(c1) VALUES (5 * 10)", "multiplication")
	test("INSERT INTO t1(c1) VALUES ('prefix_' || 'suffix')", "concatenation")

	test("WITH source AS (SELECT 1 AS id, 'test' AS name) INSERT INTO table1(id, name) SELECT id, name FROM source", "with_clause")
}
