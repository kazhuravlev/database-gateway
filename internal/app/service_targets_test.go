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
	"context"
	"sync"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/app/rules"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/stretchr/testify/require"
)

func TestServiceGetTargets(t *testing.T) {
	t.Parallel()

	targets := []config.Target{
		{
			ID:          config.TargetID("pg-1"),
			Description: "main",
			Tags:        []string{"prod"},
			Type:        "postgres",
			Connection: config.Connection{
				Host:        "",
				Port:        0,
				User:        "",
				Password:    "",
				DB:          "",
				UseSSL:      false,
				MaxPoolSize: 0,
			},
			DefaultSchema: "public",
			Tables: []config.TargetTable{
				{Table: "public.clients", Fields: nil},
			},
		},
		{
			ID:          config.TargetID("pg-2"),
			Description: "analytics",
			Tags:        []string{"analytics"},
			Type:        "postgres",
			Connection: config.Connection{
				Host:        "",
				Port:        0,
				User:        "",
				Password:    "",
				DB:          "",
				UseSSL:      false,
				MaxPoolSize: 0,
			},
			DefaultSchema: "public",
			Tables: []config.TargetTable{
				{Table: "public.events", Fields: nil},
			},
		},
	}

	testCases := []struct {
		name    string
		user    structs.User
		acls    []rules.ACL
		wantIDs []config.TargetID
	}{
		{
			name: "allow by role",
			user: structs.User{
				ID:       config.UserID("alice@example.com"),
				Username: "",
				Role:     config.RoleUser,
			},
			acls: []rules.ACL{
				{User: rules.RolePrincipal(config.RoleUser.S()), Target: "pg-1", Op: rules.Star, Tbl: rules.Star, Allow: true},
			},
			wantIDs: []config.TargetID{"pg-1"},
		},
		{
			name: "allow by user principal",
			user: structs.User{
				ID:       config.UserID("alice@example.com"),
				Username: "",
				Role:     config.RoleUser,
			},
			acls: []rules.ACL{
				{User: rules.UserPrincipal("alice@example.com"), Target: "pg-2", Op: rules.Star, Tbl: rules.Star, Allow: true},
			},
			wantIDs: []config.TargetID{"pg-2"},
		},
		{
			name: "no matching acl",
			user: structs.User{
				ID:       config.UserID("alice@example.com"),
				Username: "",
				Role:     config.RoleUser,
			},
			acls: []rules.ACL{
				{User: rules.RolePrincipal(config.RoleAdmin.S()), Target: rules.Star, Op: rules.Star, Tbl: rules.Star, Allow: true},
			},
			wantIDs: []config.TargetID{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &Service{
				opts: Options{
					logger:  nil,
					targets: targets,
					users: config.UsersProviderOIDC{
						ClientID:            "",
						ClientSecret:        "",
						IssuerURL:           "",
						RedirectURL:         "",
						Scopes:              nil,
						AccessTokenAudience: "",
						RoleClaim:           "",
						RoleMapping:         nil,
					},
					authorizer: rules.New(tc.acls),
					storage:    nil,
				},
				connsMu:       new(sync.RWMutex),
				conns:         nil,
				oauthCfg:      nil,
				oidcProvider:  nil,
				tokenVerifier: nil,
				oidcLogoutEP:  "",
				oidcRevokeEP:  "",
			}

			got, err := svc.GetTargets(context.Background(), tc.user)
			require.NoError(t, err)
			require.Len(t, got, len(tc.wantIDs))

			gotIDs := make([]config.TargetID, 0, len(got))
			for _, server := range got {
				gotIDs = append(gotIDs, server.ID)
			}

			require.Equal(t, tc.wantIDs, gotIDs)
		})
	}
}

func TestServiceGetTargetByID(t *testing.T) {
	t.Parallel()

	target := config.Target{
		ID:          config.TargetID("pg-1"),
		Description: "main",
		Tags:        []string{"prod"},
		Type:        "postgres",
		Connection: config.Connection{
			Host:        "",
			Port:        0,
			User:        "",
			Password:    "",
			DB:          "",
			UseSSL:      false,
			MaxPoolSize: 0,
		},
		DefaultSchema: "public",
		Tables: []config.TargetTable{
			{Table: "public.clients", Fields: nil},
		},
	}

	user := structs.User{
		ID:       config.UserID("alice@example.com"),
		Username: "",
		Role:     config.RoleUser,
	}

	testCases := []struct {
		name       string
		acls       []rules.ACL
		targetID   config.TargetID
		wantErr    bool
		wantErrIs  error
		wantServer *structs.Server
	}{
		{
			name: "allowed target",
			acls: []rules.ACL{
				{User: rules.RolePrincipal(config.RoleUser.S()), Target: "pg-1", Op: rules.Star, Tbl: rules.Star, Allow: true},
			},
			targetID:  config.TargetID("pg-1"),
			wantErr:   false,
			wantErrIs: nil,
			wantServer: &structs.Server{
				ID:          config.TargetID("pg-1"),
				Description: "main",
				Tags:        []structs.Tag{{Name: "prod"}},
				Type:        "postgres",
				Tables:      []config.TargetTable{{Table: "public.clients", Fields: nil}},
			},
		},
		{
			name: "target exists but forbidden",
			acls: []rules.ACL{
				{User: rules.RolePrincipal(config.RoleAdmin.S()), Target: "pg-1", Op: rules.Star, Tbl: rules.Star, Allow: true},
			},
			targetID:   config.TargetID("pg-1"),
			wantErr:    true,
			wantErrIs:  ErrNotFound,
			wantServer: nil,
		},
		{
			name: "target does not exist",
			acls: []rules.ACL{
				{User: rules.RolePrincipal(config.RoleUser.S()), Target: "pg-1", Op: rules.Star, Tbl: rules.Star, Allow: true},
			},
			targetID:   config.TargetID("pg-unknown"),
			wantErr:    true,
			wantErrIs:  ErrNotFound,
			wantServer: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &Service{
				opts: Options{
					logger:  nil,
					targets: []config.Target{target},
					users: config.UsersProviderOIDC{
						ClientID:            "",
						ClientSecret:        "",
						IssuerURL:           "",
						RedirectURL:         "",
						Scopes:              nil,
						AccessTokenAudience: "",
						RoleClaim:           "",
						RoleMapping:         nil,
					},
					authorizer: rules.New(tc.acls),
					storage:    nil,
				},
				connsMu:       new(sync.RWMutex),
				conns:         nil,
				oauthCfg:      nil,
				oidcProvider:  nil,
				tokenVerifier: nil,
				oidcLogoutEP:  "",
				oidcRevokeEP:  "",
			}

			got, err := svc.GetTargetByID(context.Background(), user, tc.targetID)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				require.ErrorIs(t, err, tc.wantErrIs)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantServer, got)
		})
	}
}

func TestGetTargetByIDReturnsSchema(t *testing.T) {
	t.Parallel()

	svc := &Service{
		opts: Options{
			logger: nil,
			targets: []config.Target{
				{
					ID:          config.TargetID("pg-1"),
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
					DefaultSchema: "public",
					Tables: []config.TargetTable{
						{Table: "public.clients", Fields: nil},
					},
				},
			},
			users: config.UsersProviderOIDC{
				ClientID:            "",
				ClientSecret:        "",
				IssuerURL:           "",
				RedirectURL:         "",
				Scopes:              nil,
				AccessTokenAudience: "",
				RoleClaim:           "",
				RoleMapping:         nil,
			},
			authorizer: rules.New([]rules.ACL{
				{User: rules.RolePrincipal(config.RoleUser.S()), Target: "pg-1", Op: rules.Star, Tbl: rules.Star, Allow: true},
			}),
			storage: nil,
		},
		connsMu:       new(sync.RWMutex),
		conns:         nil,
		oauthCfg:      nil,
		oidcProvider:  nil,
		tokenVerifier: nil,
		oidcLogoutEP:  "",
		oidcRevokeEP:  "",
	}

	_, schema, err := svc.getTargetByID(context.Background(), structs.User{
		ID:       config.UserID("alice@example.com"),
		Username: "",
		Role:     config.RoleUser,
	}, config.TargetID("pg-1"))
	require.NoError(t, err)

	_, exists := schema.GetTable("clients")
	require.True(t, exists, "default schema should allow table lookup without schema prefix")
}
