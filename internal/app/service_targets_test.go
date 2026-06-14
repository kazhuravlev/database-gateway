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
	"encoding/json"
	"log/slog"
	"sync"
	"testing"

	"github.com/go-jet/jet/v2/qrm"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/storage"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

type fakeQueryHistoryStorage struct {
	*storage.Service

	mu      sync.Mutex
	inserts []storage.InsertQueryResultsReq
}

func (*fakeQueryHistoryStorage) Conn(context.Context) qrm.DB { //nolint:ireturn
	return nil
}

func (s *fakeQueryHistoryStorage) InsertQueryResults(_ qrm.DB, req storage.InsertQueryResultsReq) error { //nolint:gocritic
	s.mu.Lock()
	defer s.mu.Unlock()

	s.inserts = append(s.inserts, req)

	return nil
}

func (s *fakeQueryHistoryStorage) inserted() []storage.InsertQueryResultsReq {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]storage.InsertQueryResultsReq(nil), s.inserts...)
}

const targetPolicy = `
package gateway

default allow_target := false
default allow_query := false

allow_target if {
	"role:user" in input.subjects
	input.target == "pg-1"
}

allow_target if {
	"user:alice@example.com" in input.subjects
	input.target == "pg-2"
}

allow_query if {
	allow_target
	input.op == "select"
}
`

func TestServiceGetTargets(t *testing.T) {
	t.Parallel()

	targets := []config.Target{
		{
			ID:          "pg-1",
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
			Tables:        []config.TargetTable{{Table: "public.clients", Fields: nil}},
		},
		{
			ID:          "pg-2",
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
			Tables:        []config.TargetTable{{Table: "public.events", Fields: nil}},
		},
	}

	testCases := []struct {
		name    string
		user    structs.User
		wantIDs []config.TargetID
	}{
		{
			name:    "allow by role",
			user:    structs.User{ID: "bob@example.com", Username: "", Role: config.RoleUser},
			wantIDs: []config.TargetID{"pg-1"},
		},
		{
			name:    "allow by user principal and role",
			user:    structs.User{ID: "alice@example.com", Username: "", Role: config.RoleUser},
			wantIDs: []config.TargetID{"pg-1", "pg-2"},
		},
		{
			name:    "no matching policy",
			user:    structs.User{ID: "admin@example.com", Username: "", Role: config.RoleAdmin},
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
					authorizer: mustAuthorizer(t, targetPolicy),
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
		ID:          "pg-1",
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
		Tables:        []config.TargetTable{{Table: "public.clients", Fields: nil}},
	}

	user := structs.User{ID: "alice@example.com", Username: "", Role: config.RoleUser}

	testCases := []struct {
		name       string
		authorizer string
		targetID   config.TargetID
		wantErrIs  error
		wantServer *structs.Server
	}{
		{
			name:       "allowed target",
			authorizer: targetPolicy,
			targetID:   "pg-1",
			wantErrIs:  nil,
			wantServer: &structs.Server{
				ID:          "pg-1",
				Description: "main",
				Tags:        []structs.Tag{{Name: "prod"}},
				Type:        "postgres",
				Tables:      []config.TargetTable{{Table: "public.clients", Fields: nil}},
			},
		},
		{
			name: "target exists but forbidden",
			authorizer: `
package gateway
default allow_target := false
default allow_query := false
`,
			targetID:   "pg-1",
			wantErrIs:  ErrNotFound,
			wantServer: nil,
		},
		{
			name:       "target does not exist",
			authorizer: targetPolicy,
			targetID:   "pg-unknown",
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
					authorizer: mustAuthorizer(t, tc.authorizer),
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
			if tc.wantErrIs != nil {
				require.ErrorIs(t, err, tc.wantErrIs)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantServer, got)
		})
	}
}

func TestPolicyReceivesCanonicalTableName(t *testing.T) {
	t.Parallel()

	schema := validator.NewDbSchema("public", []config.TargetTable{
		{Table: "public.clients", Fields: []string{"id", "name"}},
	})

	var seenTable string
	haveAccess := func(vec validator.Vec) bool {
		seenTable = schema.CanonicalTable(vec.Tbl)

		return seenTable == "public.clients"
	}

	vectors, err := validator.MakeVectors("select id, name from clients")
	require.NoError(t, err)
	require.NoError(t, validator.ValidateSchema(vectors, schema))
	require.NoError(t, validator.ValidateAccess(vectors, haveAccess))
	require.Equal(t, "public.clients", seenTable)
}

func TestRunQueryReturnsForbiddenWhenQueryPreflightDeniesAccess(t *testing.T) {
	t.Parallel()

	target := config.Target{
		ID:          "pg-1",
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
		Tables:        []config.TargetTable{{Table: "public.clients", Fields: []string{"id"}}},
	}

	testCases := []struct {
		name  string
		query string
	}{
		{
			name:  "query policy denies known table",
			query: "select id from clients",
		},
		{
			name:  "schema validation hides unknown table",
			query: "select id from unknown_table",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			history := &fakeQueryHistoryStorage{
				Service: nil,
				mu:      sync.Mutex{},
				inserts: nil,
			}
			svc := &Service{
				opts: Options{
					logger:  slog.New(slog.DiscardHandler),
					targets: []config.Target{target},
					users:   *new(config.UsersProviderOIDC),
					authorizer: mustAuthorizer(t, `
package gateway

default allow_target := false
default allow_query := false

allow_target if {
	"role:user" in input.subjects
	input.target == "pg-1"
}
`),
					storage: history,
				},
				connsMu:       new(sync.RWMutex),
				conns:         nil,
				oauthCfg:      nil,
				oidcProvider:  nil,
				tokenVerifier: nil,
				oidcLogoutEP:  "",
				oidcRevokeEP:  "",
			}

			_, err := svc.RunQuery(
				context.Background(),
				structs.User{ID: "alice@example.com", Username: "alice", Role: config.RoleUser},
				"pg-1",
				tc.query,
			)

			require.ErrorIs(t, err, ErrForbidden)

			inserted := history.inserted()
			require.Len(t, inserted, 1)

			item := inserted[0]
			require.Equal(t, config.UserID("alice@example.com"), item.UserID)
			require.Equal(t, config.TargetID("pg-1"), item.TargetID)
			require.Equal(t, tc.query, item.Query)

			var payload storedQueryResultPayload
			require.NoError(t, json.Unmarshal(item.Response, &payload))
			require.Equal(t, "failed", payload.Status)
			require.Equal(t, "access denied", payload.Error)
		})
	}
}
