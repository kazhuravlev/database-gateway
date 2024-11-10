package validator_test

import (
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMakeVectors(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		f := func(name, query string, exp []validator.Vec) {
			t.Run(name, func(t *testing.T) {
				vecs, err := validator.MakeVectors(query)
				require.NoError(t, err)
				require.Equal(t, exp, vecs)
			})
		}

		f("select_complex",
			`select f1, count(f2) from clients where f3=1 group by f4 order by f5`,
			[]validator.Vec{{
				Op:   config.OpSelect,
				Tbl:  "public.clients",
				Cols: []string{"f1", "f2", "f3", "f4", "f5"},
			}})
		f("insert_complex",
			`insert into clients (f1, f2) values (1, 2) on conflict (f3, f4) do update set f5=33 returning f6`,
			[]validator.Vec{{
				Op:   config.OpInsert,
				Tbl:  "public.clients",
				Cols: []string{"f1", "f2", "f3", "f4", "f5", "f6"},
			}})
		f("update_complex",
			`update clients set f1=1 where f2=2 returning f3`,
			[]validator.Vec{{
				Op:   config.OpUpdate,
				Tbl:  "public.clients",
				Cols: []string{"f1", "f2", "f3"},
			}})
		f("delete_complex",
			`delete from clients where f1=1 returning f2`,
			[]validator.Vec{{
				Op:   config.OpDelete,
				Tbl:  "public.clients",
				Cols: []string{"f1", "f2"},
			}})
	})

	t.Run("bad_path", func(t *testing.T) {
		f := func(name, query string, err error) {
			t.Run(name, func(t *testing.T) {
				vecs, err2 := validator.MakeVectors(query)
				require.Error(t, err2)
				require.ErrorIs(t, err2, err)
				require.Nil(t, vecs)
			})
		}

		f("select_star",
			`select * from clients`,
			validator.ErrComplicatedQuery)
		f("insert_star",
			`insert into clients(f1, f3) values(1,2) returning *`,
			validator.ErrComplicatedQuery)
		f("update_star",
			`update clients set f1=1 where f2=2 returning *`,
			validator.ErrComplicatedQuery)
		f("delete_star",
			`delete from clients where f1=1 returning *`,
			validator.ErrComplicatedQuery)
	})
}
