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

		t.Run("query_has_several_statements", func(t *testing.T) {
			query := `select 1; select 1`
			err := validator.IsAllowed(config.Target{}, config.User{}, query)
			require.Error(t, err)
		})
	})
}
