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
			`insert into clients (f1, f2) values (1, 2) returning f3`,
			[]validator.Vec{{
				Op:   config.OpInsert,
				Tbl:  "public.clients",
				Cols: []string{"f1", "f2", "f3"},
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
}
