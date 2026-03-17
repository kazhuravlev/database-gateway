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

package opa_test

import (
	"context"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/policy/opa"
	"github.com/stretchr/testify/require"
)

// ExamplePolicySimple shows the minimum shape expected by the embedded OPA authorizer.
const ExamplePolicySimple = `
package gateway

default allow_target := false
default allow_query := false

allow_target if {
	"role:user" in input.subjects
	input.target == "local-1"
}

allow_query if {
	"role:user" in input.subjects
	input.target == "local-1"
	input.op == "select"
	input.table == "public.clients"
}
`

func TestAuthorizerAllowTarget(t *testing.T) {
	t.Parallel()

	authz, err := opa.New(context.Background(), map[string]string{
		"example.rego": ExamplePolicySimple,
	})
	require.NoError(t, err)

	require.True(t, authz.AllowTarget([]string{"user:alice@example.com", "role:user"}, "local-1"))
	require.False(t, authz.AllowTarget([]string{"user:alice@example.com", "role:user"}, "local-2"))
	require.False(t, authz.AllowTarget([]string{"user:admin@example.com", "role:admin"}, "local-1"))
}

func TestAuthorizerAllowQuery(t *testing.T) {
	t.Parallel()

	authz, err := opa.New(context.Background(), map[string]string{
		"example.rego": ExamplePolicySimple,
	})
	require.NoError(t, err)

	require.True(t, authz.AllowQuery(
		[]string{"user:alice@example.com", "role:user"},
		"local-1",
		"select",
		"public.clients",
	))
	require.False(t, authz.AllowQuery(
		[]string{"user:alice@example.com", "role:user"},
		"local-1",
		"update",
		"public.clients",
	))
	require.False(t, authz.AllowQuery(
		[]string{"user:alice@example.com", "role:user"},
		"local-1",
		"select",
		"public.orders",
	))
}

func TestNewFailsForInvalidModule(t *testing.T) {
	t.Parallel()

	authz, err := opa.New(context.Background(), map[string]string{
		"broken.rego": "package gateway\nallow_target if { this is bad }",
	})
	require.Error(t, err)
	require.Nil(t, authz)
}
