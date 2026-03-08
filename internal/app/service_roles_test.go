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

package app //nolint:testpackage

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/app/rules"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/storage"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/stretchr/testify/require"
)

func TestResolveUserRole(t *testing.T) {
	t.Parallel()

	const roleClaimGroups = "groups"
	const roleClaimDepartment = "department"
	testCases := []struct {
		name           string
		claims         map[string]any
		cfgRoleClaim   string
		cfgRoleMapping map[string]config.Role
		wantRole       config.Role
		wantErr        bool
	}{
		{
			name: "matched group",
			claims: map[string]any{
				roleClaimGroups: []string{"dbgw-users", "ops"},
			},
			cfgRoleClaim: roleClaimGroups,
			cfgRoleMapping: map[string]config.Role{
				"dbgw-admins": config.RoleAdmin,
				"dbgw-users":  config.RoleUser,
			},
			wantRole: config.RoleUser,
			wantErr:  false,
		},
		{
			name: "no fallbacks",
			claims: map[string]any{
				roleClaimGroups: []string{"unknown"},
			},
			cfgRoleClaim: roleClaimGroups,
			cfgRoleMapping: map[string]config.Role{
				"dbgw-admins": config.RoleAdmin,
			},
			wantRole: "",
			wantErr:  true,
		},
		{
			name: "single string claim not supported",
			claims: map[string]any{
				roleClaimDepartment: "platform-admins",
			},
			cfgRoleClaim: roleClaimDepartment,
			cfgRoleMapping: map[string]config.Role{
				"platform-admins": config.RoleAdmin,
			},
			wantRole: "",
			wantErr:  true,
		},
		{
			name: "invalid claim type",
			claims: map[string]any{
				roleClaimGroups: map[string]string{"name": "dbgw-users"},
			},
			cfgRoleClaim: roleClaimGroups,
			cfgRoleMapping: map[string]config.Role{
				"dbgw-users": config.RoleUser,
			},
			wantRole: "",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rawClaims, err := toRawClaims(tc.claims)
			require.NoError(t, err)

			gotRole, err := resolveUserRole(rawClaims, tc.cfgRoleClaim, tc.cfgRoleMapping)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			require.NoError(t, err)

			if gotRole != tc.wantRole {
				t.Fatalf("unexpected role: got=%q want=%q", gotRole, tc.wantRole)
			}
		})
	}
}

func TestUserSubjectsIncludesRolePrincipal(t *testing.T) {
	t.Parallel()

	subjects := userSubjects(structs.User{
		ID:       "user@example.com",
		Username: "user",
		Role:     config.RoleAdmin,
	})

	require.Len(t, subjects, 2)
	require.Equal(t, []string{
		"user:user@example.com",
		"role:admin",
	}, subjects)
}

func TestGetClaimValues(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()

		values, err := getClaimValues(map[string]json.RawMessage{
			"groups": json.RawMessage(`["dbgw-users","ops"]`),
		}, "groups")
		require.NoError(t, err)
		require.Equal(t, []string{"dbgw-users", "ops"}, values)
	})

	t.Run("claim missing", func(t *testing.T) {
		t.Parallel()

		values, err := getClaimValues(map[string]json.RawMessage{}, "groups")
		require.Error(t, err)
		require.Nil(t, values)
	})

	t.Run("invalid claim type", func(t *testing.T) {
		t.Parallel()

		values, err := getClaimValues(map[string]json.RawMessage{
			"groups": json.RawMessage(`"dbgw-users"`),
		}, "groups")
		require.Error(t, err)
		require.Nil(t, values)
	})
}

func TestNewRequiresRoleMapping(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)

	opts := NewOptions(
		logger,
		[]config.Target{
			{
				ID:          "pg-1",
				Description: "",
				Tags:        nil,
				Type:        "",
				Connection: config.Connection{
					Host:        "",
					Port:        0,
					User:        "",
					Password:    "",
					DB:          "",
					UseSSL:      false,
					MaxPoolSize: 0,
				},
				DefaultSchema: "",
				Tables:        nil,
			},
		},
		config.UsersProviderOIDC{
			ClientID:     "cid",
			ClientSecret: "secret",
			IssuerURL:    "http://localhost:9000/application/o/db-gateway/",
			RedirectURL:  "http://localhost:8080/auth/callback",
			Scopes:       []string{"openid"},
			RoleClaim:    "groups",
			RoleMapping:  map[string]config.Role{},
		},
		rules.New([]rules.ACL{}),
		&storage.Service{},
	)

	service, err := New(opts)
	require.Error(t, err)
	require.Nil(t, service)
	require.ErrorContains(t, err, "no role mappings defined")
}

func toRawClaims(claims map[string]any) (map[string]json.RawMessage, error) {
	rawClaims := make(map[string]json.RawMessage, len(claims))
	for claim, value := range claims {
		buf, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}

		rawClaims[claim] = buf
	}

	return rawClaims, nil
}
