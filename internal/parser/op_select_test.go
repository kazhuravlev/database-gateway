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

func TestParseSelectValid(t *testing.T) {
	t.Parallel()

	fValid := func(input string) {
		t.Helper()
		t.Run("", func(t *testing.T) {
			t.Helper()
			_, err := parser.Parse(input)
			require.NoError(t, err)
		})
	}

	fValid("SELECT public.clients.id from public.clients")
	fValid("SELECT clients.id from public.clients")
	fValid("SELECT id from public.clients")

	fValid("SELECT public.clients.id from clients")
	fValid("SELECT clients.id from clients")
	fValid("SELECT id from clients")

	fValid("SELECT id, id from clients")
	fValid(`SELECT "id" from "clients"`)

	fValid(`SELECT public.clients.c1, clients.c2, c3 from "clients"`)
	fValid(`SELECT clients.id AS client_id from clients`)
	fValid(`SELECT "public"."clients"."id" from "public"."clients"`)

	fValid(`SELECT id FROM clients WHERE id > 0`)
	fValid(`SELECT id FROM clients WHERE clients.id > 0`)
	fValid(`SELECT id FROM clients WHERE id = 42`)
	fValid(`SELECT id FROM clients WHERE id != 42`)
	fValid(`SELECT id FROM clients WHERE id > 42`)
	fValid(`SELECT id FROM clients WHERE id < 42`)
	fValid(`SELECT id FROM clients WHERE id is null`)
	fValid(`SELECT id FROM clients WHERE id is NULL`)
	fValid(`SELECT id FROM clients WHERE id is not NULL`)

	fValid(`SELECT id FROM clients ORDER BY id ASC`)
	fValid(`SELECT id FROM clients ORDER BY id DESC`)
	fValid(`SELECT id FROM clients LIMIT 10`)
	fValid(`SELECT id FROM clients OFFSET 5`)
	fValid(`SELECT id FROM clients LIMIT 10 OFFSET 5`)

	fValid(`SELECT c1, c2 FROM clients GROUP BY c1, c2`)
	fValid(`SELECT c1 FROM clients GROUP BY c1`)
	fValid(`SELECT c1 FROM clients GROUP BY clients.c1`)
	fValid(`SELECT c1 FROM clients GROUP BY public.clients.c1`)
	fValid(`SELECT c1 FROM clients GROUP BY "public"."clients"."c1"`)

	fValid(`SELECT c1 FROM "schema-with-dash"."table-with-dash"`)
	fValid(`SELECT c1 FROM schema1.clients`)

	fValid(`SELECT lower(id) from clients`)
	fValid(`SELECT count(*) from clients`)
}

// TestParseSelectInvalid is just a list of special cases that is not supported (yet).
func TestParseSelectInvalid(t *testing.T) {
	t.Parallel()

	fInvalid := func(input, name string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()
			_, err := parser.Parse(input)
			require.Error(t, err)
		})
	}

	fInvalid("SELECT * FROM clients", "star_expression")
	fInvalid("SELECT clients.* FROM clients", "star_expression")
	fInvalid("SELECT public.clients.* FROM clients", "star_expression")

	fInvalid("SELECT * FROM public.clients", "star_expression_scm")
	fInvalid("SELECT clients.* FROM public.clients", "star_expression_scm")
	fInvalid("SELECT public.clients.* FROM public.clients", "star_expression_scm")

	fInvalid("SELECT id FROM clients INNER JOIN orders ON clients.id = orders.client_id", "join_expression")
	fInvalid("SELECT id FROM (SELECT id FROM clients) AS sub", "nested_select")
	fInvalid("SELECT id FROM clients UNION SELECT id FROM orders", "union_expression")
}
