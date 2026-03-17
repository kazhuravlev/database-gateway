package app

import (
	"context"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/policy/opa"
	"github.com/stretchr/testify/require"
)

func mustAuthorizer(t *testing.T, module string) *opa.Authorizer {
	t.Helper()

	authz, err := opa.New(context.Background(), map[string]string{
		"policy.rego": module,
	})
	require.NoError(t, err)

	return authz
}
