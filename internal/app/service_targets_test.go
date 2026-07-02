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
	"github.com/kazhuravlev/database-gateway/internal/storage/jetgen/model"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/stretchr/testify/require"
)

type fakeQueryHistoryStorage struct {
	mu                 sync.Mutex
	inserts            []storage.InsertQueryResultsReq
	states             []storage.SetQueryResultsStateReq
	queryResult        *model.QueryResults
	queryResultErr     error
	queryResultIDs     []uuid6.UUID
	listByUserItems    []storage.QueryResult
	listByUserErr      error
	listByUserLimits   []int64
	listRequestsItems  []storage.QueryResult
	listRequestsErr    error
	listRequestsLimits []int64
	listRequestsOffset []int64
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

func (s *fakeQueryHistoryStorage) SetQueryResultsState(_ qrm.DB, req storage.SetQueryResultsStateReq) error { //nolint:gocritic
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states = append(s.states, req)

	return nil
}

func (s *fakeQueryHistoryStorage) GetQueryResultsByID(_ qrm.DB, queryID uuid6.UUID) (*model.QueryResults, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queryResultIDs = append(s.queryResultIDs, queryID)
	if s.queryResultErr != nil {
		return nil, s.queryResultErr
	}
	if s.queryResult == nil {
		return nil, storage.ErrNotFound
	}

	res := *s.queryResult
	res.Response = append([]byte(nil), s.queryResult.Response...)

	return &res, nil
}

func (s *fakeQueryHistoryStorage) ListQueryResultsByUser(
	_ qrm.DB,
	_ config.UserID,
	limit int64,
) ([]storage.QueryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.listByUserLimits = append(s.listByUserLimits, limit)
	if s.listByUserErr != nil {
		return nil, s.listByUserErr
	}

	return append([]storage.QueryResult(nil), s.listByUserItems...), nil
}

func (s *fakeQueryHistoryStorage) ListQueryResults(_ qrm.DB, limit, offset int64) ([]storage.QueryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.listRequestsLimits = append(s.listRequestsLimits, limit)
	s.listRequestsOffset = append(s.listRequestsOffset, offset)
	if s.listRequestsErr != nil {
		return nil, s.listRequestsErr
	}

	return append([]storage.QueryResult(nil), s.listRequestsItems...), nil
}

func (*fakeQueryHistoryStorage) InsertBookmark(qrm.DB, storage.InsertBookmarkReq) error {
	return nil
}

func (*fakeQueryHistoryStorage) DeleteBookmark(qrm.DB, config.UserID, uuid6.UUID) error {
	return nil
}

func (*fakeQueryHistoryStorage) ListBookmarks(qrm.DB, config.UserID, config.TargetID) ([]storage.Bookmark, error) {
	return []storage.Bookmark{}, nil
}

func (*fakeQueryHistoryStorage) ListBookmarksByUser(qrm.DB, config.UserID) ([]storage.Bookmark, error) {
	return []storage.Bookmark{}, nil
}

func (s *fakeQueryHistoryStorage) inserted() []storage.InsertQueryResultsReq {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]storage.InsertQueryResultsReq(nil), s.inserts...)
}

func (s *fakeQueryHistoryStorage) statesSet() []storage.SetQueryResultsStateReq {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]storage.SetQueryResultsStateReq(nil), s.states...)
}

func (s *fakeQueryHistoryStorage) queryResultIDsSet() []uuid6.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]uuid6.UUID(nil), s.queryResultIDs...)
}

func (s *fakeQueryHistoryStorage) listByUserLimitsSet() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]int64(nil), s.listByUserLimits...)
}

func (s *fakeQueryHistoryStorage) listRequestsLimitsSet() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]int64(nil), s.listRequestsLimits...)
}

func (s *fakeQueryHistoryStorage) listRequestsOffsetsSet() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]int64(nil), s.listRequestsOffset...)
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
		name      string
		query     string
		wantError string
	}{
		{
			name:      "query policy denies known table",
			query:     "select id from clients",
			wantError: "preflight check: validate access: forbidden",
		},
		{
			name:      "schema validation hides unknown table",
			query:     "select id from unknown_table",
			wantError: "preflight check: validate schema: forbidden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			history := &fakeQueryHistoryStorage{
				mu:      sync.Mutex{},
				inserts: nil,
				states:  nil,
			}
			svc := &Service{
				opts: Options{
					logger:  slog.New(slog.DiscardHandler),
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

			queryID, err := svc.RunQuery(
				context.Background(),
				structs.User{ID: "alice@example.com", Username: "alice", Role: config.RoleUser},
				"pg-1",
				tc.query,
			)

			require.NoError(t, err)

			inserted := history.inserted()
			require.Len(t, inserted, 1)

			item := inserted[0]
			require.Equal(t, queryID, item.ID)
			require.Equal(t, config.UserID("alice@example.com"), item.UserID)
			require.Equal(t, config.TargetID("pg-1"), item.TargetID)
			require.Equal(t, tc.query, item.Query)

			statesSet := history.statesSet()
			require.Len(t, statesSet, 1)
			require.Equal(t, item.ID, statesSet[0].ID)
			require.Equal(t, structs.QueryStateFailed, statesSet[0].State)

			var payload structs.QError
			require.NoError(t, json.Unmarshal(statesSet[0].Payload, &payload))
			require.Equal(t, tc.wantError, payload.Error)
		})
	}
}
