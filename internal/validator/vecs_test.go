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

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

func TestMakeVectors(t *testing.T) {
	t.Parallel()

	t.Run("happy_path", func(t *testing.T) {
		t.Parallel()
		test := func(name, query string, exp []validator.Vec) {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				vecs, err := validator.MakeVectors(query)
				require.NoError(t, err)
				require.Equal(t, exp, vecs)
			})
		}

		test("select_complex",
			`select f1, count(f2) from clients where f3=1 group by f4 order by f5`,
			[]validator.Vec{{
				Op:   config.OpSelect,
				Tbl:  "public.clients",
				Cols: []string{"f1", "f2", "f3", "f4", "f5"},
			}})
		test("insert_complex",
			`insert into clients (f1, f2) values (1, 2) on conflict (f3, f4) do update set f5=33 returning f6`,
			[]validator.Vec{{
				Op:   config.OpInsert,
				Tbl:  "public.clients",
				Cols: []string{"f1", "f2", "f3", "f4", "f5", "f6"},
			}})
		test("update_complex",
			`update clients set f1=1 where f2=2 returning f3`,
			[]validator.Vec{{
				Op:   config.OpUpdate,
				Tbl:  "public.clients",
				Cols: []string{"f1", "f2", "f3"},
			}})
		test("delete_complex",
			`delete from clients where f1=1 returning f2`,
			[]validator.Vec{{
				Op:   config.OpDelete,
				Tbl:  "public.clients",
				Cols: []string{"f1", "f2"},
			}})
	})

	t.Run("bad_path", func(t *testing.T) {
		t.Parallel()
		test := func(name, query string, err error) {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				vecs, err2 := validator.MakeVectors(query)
				require.Error(t, err2)
				require.ErrorIs(t, err2, err)
				require.Nil(t, vecs)
			})
		}

		test("select_star",
			`select * from clients`,
			validator.ErrComplicatedQuery)
		test("insert_star",
			`insert into clients(f1, f3) values(1,2) returning *`,
			validator.ErrComplicatedQuery)
		test("update_star",
			`update clients set f1=1 where f2=2 returning *`,
			validator.ErrComplicatedQuery)
		test("delete_star",
			`delete from clients where f1=1 returning *`,
			validator.ErrComplicatedQuery)
	})
}
