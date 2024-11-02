package validator_test

import (
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	t.Run("bad_requests", func(t *testing.T) {
		t.Run("query_has_no_statements", func(t *testing.T) {
			query := ``
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})

		t.Run("not_a_query", func(t *testing.T) {
			query := `what time is it?`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})

		t.Run("query_has_several_statements", func(t *testing.T) {
			query := `select 1; select 1`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})

		t.Run("star_select", func(t *testing.T) {
			query := `select * from table`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})

		t.Run("schema_changes", func(t *testing.T) {
			query := `create table aaa(id text);`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})

		t.Run("alter_table", func(t *testing.T) {
			query := `alter table aaa add column id text default '';`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})
	})

	t.Run("positive_cases", func(t *testing.T) {
		t.Run("asdasd", func(t *testing.T) {
			target := config.Target{Id: "t1"}
			acls := []config.ACL{{
				Op:     config.OpSelect,
				Target: "t1",
				Tbl:    "clients",
				Allow:  true,
			}}
			query := `select id, name from clients;`
			err := validator.IsAllowed(target, config.User{Acls: acls}, query)
			require.NoError(t, err)
		})
	})
}
