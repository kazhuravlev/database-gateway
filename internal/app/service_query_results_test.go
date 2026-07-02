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

package app //nolint:exhaustruct,testpackage

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/storage"
	"github.com/kazhuravlev/database-gateway/internal/storage/jetgen/model"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
	"github.com/stretchr/testify/require"
)

var errQueryResultsStorageFailed = errors.New("storage failed")

func TestCanReadQueryResults(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		user    structs.User
		ownerID config.UserID
		want    bool
	}{
		{
			name: "owner can read own results",
			user: structs.User{
				ID:       config.UserID("alice@example.com"),
				Username: "",
				Role:     config.RoleUser,
			},
			ownerID: config.UserID("alice@example.com"),
			want:    true,
		},
		{
			name: "non owner user cannot read another users results",
			user: structs.User{
				ID:       config.UserID("bob@example.com"),
				Username: "",
				Role:     config.RoleUser,
			},
			ownerID: config.UserID("alice@example.com"),
			want:    false,
		},
		{
			name: "admin can read any results",
			user: structs.User{
				ID:       config.UserID("admin@example.com"),
				Username: "",
				Role:     config.RoleAdmin,
			},
			ownerID: config.UserID("alice@example.com"),
			want:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, canReadQueryResults(tc.user, tc.ownerID))
		})
	}
}

func TestServiceGetQueryResults(t *testing.T) {
	t.Parallel()

	queryResultID := uuid6.New()
	createdAt := time.Date(2026, 7, 2, 12, 30, 0, 0, time.UTC)
	owner := structs.User{
		ID:       config.UserID("alice@example.com"),
		Username: "alice",
		Role:     config.RoleUser,
	}
	otherUser := structs.User{
		ID:       config.UserID("bob@example.com"),
		Username: "bob",
		Role:     config.RoleUser,
	}
	table := structs.QTable{
		Headers: []string{"id", "name"},
		Rows:    [][]string{{"1", "Alice"}},
	}
	meta := structs.QMeta{
		RowsCount:    1,
		ColumnsCount: 2,
	}

	completedPayload, err := json.Marshal(storedQueryResultPayload{
		Table: table,
		Meta:  meta,
	})
	require.NoError(t, err)
	failedPayload, err := json.Marshal(structs.QError{Error: "query failed"})
	require.NoError(t, err)

	baseResult := model.QueryResults{
		ID:        queryResultID,
		UserID:    owner.ID,
		CreatedAt: createdAt,
		Query:     "select 1",
		Response:  completedPayload,
		TargetID:  config.TargetID("pg-1"),
		State:     structs.QueryStateCompleted,
	}

	testCases := []struct {
		name      string
		user      structs.User
		result    model.QueryResults
		wantTable structs.QTable
		wantMeta  structs.QMeta
		wantError structs.QError
		errIs     error
		errText   string
	}{
		{
			name:      "completed payload unmarshalling",
			user:      owner,
			result:    baseResult,
			wantTable: table,
			wantMeta:  meta,
		},
		{
			name: "failed payload unmarshalling",
			user: owner,
			result: model.QueryResults{
				ID:        queryResultID,
				UserID:    owner.ID,
				CreatedAt: createdAt,
				Query:     "select broken",
				Response:  failedPayload,
				TargetID:  config.TargetID("pg-1"),
				State:     structs.QueryStateFailed,
			},
			wantError: structs.QError{Error: "query failed"},
		},
		{
			name: "malformed json errors",
			user: owner,
			result: model.QueryResults{
				ID:        queryResultID,
				UserID:    owner.ID,
				CreatedAt: createdAt,
				Query:     "select 1",
				Response:  []byte("{"),
				TargetID:  config.TargetID("pg-1"),
				State:     structs.QueryStateCompleted,
			},
			errText: "unmarshal query results",
		},
		{
			name:      "owner access",
			user:      owner,
			result:    baseResult,
			wantTable: table,
			wantMeta:  meta,
		},
		{
			name:    "non-owner denial",
			user:    otherUser,
			result:  baseResult,
			errIs:   ErrNotFound,
			errText: "user does not have access to this query result",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &fakeQueryHistoryStorage{queryResult: &tc.result}
			svc := &Service{opts: Options{storage: store}}

			got, err := svc.GetQueryResults(context.Background(), tc.user, queryResultID)

			require.Equal(t, []uuid6.UUID{queryResultID}, store.queryResultIDsSet())
			if tc.errIs != nil || tc.errText != "" {
				require.Nil(t, got)
				if tc.errIs != nil {
					require.ErrorIs(t, err, tc.errIs)
				} else {
					require.Error(t, err)
				}
				if tc.errText != "" {
					require.ErrorContains(t, err, tc.errText)
				}

				return
			}

			require.NoError(t, err)
			require.Equal(t, queryResultID.S(), got.ID)
			require.Equal(t, tc.result.UserID.S(), got.UserID)
			require.Equal(t, tc.result.TargetID.S(), got.TargetID)
			require.Equal(t, tc.result.CreatedAt, got.CreatedAt)
			require.Equal(t, tc.result.Query, got.Query)
			require.Equal(t, tc.result.State, got.State)

			if tc.wantTable.Headers != nil {
				require.True(t, got.QTable.HasVal())
				require.Equal(t, tc.wantTable, got.QTable.Val())
				require.Equal(t, tc.wantMeta, got.Meta)
			} else {
				require.False(t, got.QTable.HasVal())
			}
			if tc.wantError.Error != "" {
				require.True(t, got.QError.HasVal())
				require.Equal(t, tc.wantError, got.QError.Val())
			} else {
				require.False(t, got.QError.HasVal())
			}
		})
	}
}

func TestListRecentQueriesValidatesLimit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		limit int64
	}{
		{name: "zero", limit: 0},
		{name: "negative", limit: -1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := new(fakeQueryHistoryStorage)
			svc := &Service{opts: Options{storage: store}}

			got, err := svc.ListRecentQueries(context.Background(), config.UserID("alice@example.com"), tc.limit)

			require.Nil(t, got)
			require.ErrorIs(t, err, errInvalidArgument)
			require.Empty(t, store.listByUserLimitsSet())
		})
	}
}

func TestListRecentQueriesPassesLimit(t *testing.T) {
	t.Parallel()

	store := new(fakeQueryHistoryStorage)
	svc := &Service{opts: Options{storage: store}}

	got, err := svc.ListRecentQueries(context.Background(), config.UserID("alice@example.com"), 7)

	require.NoError(t, err)
	require.Empty(t, got)
	require.Equal(t, []int64{7}, store.listByUserLimitsSet())
}

func TestListRecentQueriesStorageError(t *testing.T) {
	t.Parallel()

	errStorage := errQueryResultsStorageFailed
	store := &fakeQueryHistoryStorage{listByUserErr: errStorage}
	svc := &Service{opts: Options{storage: store}}

	got, err := svc.ListRecentQueries(context.Background(), config.UserID("alice@example.com"), 7)

	require.Nil(t, got)
	require.ErrorIs(t, err, errStorage)
	require.ErrorContains(t, err, "list query results by user")
	require.Equal(t, []int64{7}, store.listByUserLimitsSet())
}

func TestListRecentQueriesSkipsMalformedPayloads(t *testing.T) {
	t.Parallel()

	completedPayload, err := json.Marshal(storedQueryResultPayload{})
	require.NoError(t, err)

	validID := uuid6.New()
	store := &fakeQueryHistoryStorage{
		listByUserItems: []storage.QueryResult{
			{
				ID:        uuid6.New(),
				UserID:    config.UserID("alice@example.com"),
				TargetID:  config.TargetID("pg-1"),
				CreatedAt: time.Date(2026, 7, 2, 12, 30, 0, 0, time.UTC),
				Query:     "select broken",
				State:     structs.QueryStateCompleted,
				Response:  []byte("{"),
			},
			{
				ID:        validID,
				UserID:    config.UserID("alice@example.com"),
				TargetID:  config.TargetID("pg-2"),
				CreatedAt: time.Date(2026, 7, 2, 12, 31, 0, 0, time.UTC),
				Query:     "select 1",
				State:     structs.QueryStateCompleted,
				Response:  completedPayload,
			},
		},
	}
	svc := &Service{opts: Options{storage: store}}

	got, err := svc.ListRecentQueries(context.Background(), config.UserID("alice@example.com"), 7)

	require.NoError(t, err)
	require.Equal(t, []structs.Query{
		{
			ID:        validID.S(),
			TargetID:  config.TargetID("pg-2"),
			Query:     "select 1",
			State:     structs.QueryStateCompleted,
			CreatedAt: "2026-07-02 12:31:00",
		},
	}, got)
}

func TestListRecentQueriesMixedStates(t *testing.T) {
	t.Parallel()

	completedPayload, err := json.Marshal(storedQueryResultPayload{})
	require.NoError(t, err)
	failedPayload, err := json.Marshal(structs.QError{Error: "query failed"})
	require.NoError(t, err)

	completedID := uuid6.New()
	failedID := uuid6.New()
	pendingID := uuid6.New()
	store := &fakeQueryHistoryStorage{
		listByUserItems: []storage.QueryResult{
			{
				ID:        completedID,
				UserID:    config.UserID("alice@example.com"),
				TargetID:  config.TargetID("pg-1"),
				CreatedAt: time.Date(2026, 7, 2, 12, 30, 0, 0, time.UTC),
				Query:     "select 1",
				State:     structs.QueryStateCompleted,
				Response:  completedPayload,
			},
			{
				ID:        failedID,
				UserID:    config.UserID("alice@example.com"),
				TargetID:  config.TargetID("pg-2"),
				CreatedAt: time.Date(2026, 7, 2, 12, 31, 0, 0, time.UTC),
				Query:     "select broken",
				State:     structs.QueryStateFailed,
				Response:  failedPayload,
			},
			{
				ID:        pendingID,
				UserID:    config.UserID("alice@example.com"),
				TargetID:  config.TargetID("pg-3"),
				CreatedAt: time.Date(2026, 7, 2, 12, 32, 0, 0, time.UTC),
				Query:     "select pending",
				State:     structs.QueryStateNew,
				Response:  nil,
			},
		},
	}
	svc := &Service{opts: Options{storage: store}}

	got, err := svc.ListRecentQueries(context.Background(), config.UserID("alice@example.com"), 7)

	require.NoError(t, err)
	require.Equal(t, []structs.Query{
		{
			ID:        completedID.S(),
			TargetID:  config.TargetID("pg-1"),
			Query:     "select 1",
			State:     structs.QueryStateCompleted,
			CreatedAt: "2026-07-02 12:30:00",
		},
		{
			ID:        failedID.S(),
			TargetID:  config.TargetID("pg-2"),
			Query:     "select broken",
			State:     structs.QueryStateFailed,
			CreatedAt: "2026-07-02 12:31:00",
		},
		{
			ID:        pendingID.S(),
			TargetID:  config.TargetID("pg-3"),
			Query:     "select pending",
			State:     structs.QueryStateNew,
			CreatedAt: "2026-07-02 12:32:00",
		},
	}, got)
	require.Equal(t, []int64{7}, store.listByUserLimitsSet())
}

func TestListAdminRequestsValidatesPagination(t *testing.T) {
	t.Parallel()

	admin := structs.User{ID: config.UserID("admin@example.com"), Username: "admin", Role: config.RoleAdmin}
	testCases := []struct {
		name     string
		page     int64
		pageSize int64
	}{
		{name: "zero page", page: 0, pageSize: 50},
		{name: "negative page", page: -1, pageSize: 50},
		{name: "zero page size", page: 1, pageSize: 0},
		{name: "negative page size", page: 1, pageSize: -1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := new(fakeQueryHistoryStorage)
			svc := &Service{opts: Options{storage: store}}

			got, hasNext, err := svc.ListAdminRequests(context.Background(), admin, tc.page, tc.pageSize)

			require.Nil(t, got)
			require.False(t, hasNext)
			require.ErrorIs(t, err, errInvalidArgument)
			require.Empty(t, store.listRequestsLimitsSet())
			require.Empty(t, store.listRequestsOffsetsSet())
		})
	}
}

func TestListAdminRequestsDeniesNonAdmin(t *testing.T) {
	t.Parallel()

	store := new(fakeQueryHistoryStorage)
	svc := &Service{opts: Options{storage: store}}
	user := structs.User{ID: config.UserID("alice@example.com"), Username: "alice", Role: config.RoleUser}

	got, hasNext, err := svc.ListAdminRequests(context.Background(), user, 1, 50)

	require.Nil(t, got)
	require.False(t, hasNext)
	require.ErrorIs(t, err, ErrForbidden)
	require.Empty(t, store.listRequestsLimitsSet())
	require.Empty(t, store.listRequestsOffsetsSet())
}

func TestListAdminRequestsPassesPagination(t *testing.T) {
	t.Parallel()

	store := new(fakeQueryHistoryStorage)
	svc := &Service{opts: Options{storage: store}}
	admin := structs.User{ID: config.UserID("admin@example.com"), Username: "admin", Role: config.RoleAdmin}

	got, hasNext, err := svc.ListAdminRequests(context.Background(), admin, 3, 25)

	require.NoError(t, err)
	require.False(t, hasNext)
	require.Empty(t, got)
	require.Equal(t, []int64{26}, store.listRequestsLimitsSet())
	require.Equal(t, []int64{50}, store.listRequestsOffsetsSet())
}

func TestListAdminRequestsTrimsHasNextItem(t *testing.T) {
	t.Parallel()

	firstID := uuid6.New()
	secondID := uuid6.New()
	store := &fakeQueryHistoryStorage{
		listRequestsItems: []storage.QueryResult{
			{
				ID:        firstID,
				UserID:    config.UserID("alice@example.com"),
				TargetID:  config.TargetID("pg-1"),
				Query:     "select 1",
				CreatedAt: time.Date(2026, 7, 2, 12, 30, 0, 0, time.UTC),
			},
			{
				ID:        secondID,
				UserID:    config.UserID("bob@example.com"),
				TargetID:  config.TargetID("pg-2"),
				Query:     "select 2",
				CreatedAt: time.Date(2026, 7, 2, 12, 31, 0, 0, time.UTC),
			},
		},
	}
	svc := &Service{opts: Options{storage: store}}
	admin := structs.User{ID: config.UserID("admin@example.com"), Username: "admin", Role: config.RoleAdmin}

	got, hasNext, err := svc.ListAdminRequests(context.Background(), admin, 1, 1)

	require.NoError(t, err)
	require.True(t, hasNext)
	require.Equal(t, []structs.AdminRequest{
		{
			ID:        firstID.S(),
			UserID:    config.UserID("alice@example.com"),
			TargetID:  config.TargetID("pg-1"),
			Query:     "select 1",
			CreatedAt: "2026-07-02 12:30:00",
		},
	}, got)
	require.Equal(t, []int64{2}, store.listRequestsLimitsSet())
	require.Equal(t, []int64{0}, store.listRequestsOffsetsSet())
}

func TestListAdminRequestsPropagatesStorageError(t *testing.T) {
	t.Parallel()

	errStorage := errQueryResultsStorageFailed
	store := &fakeQueryHistoryStorage{listRequestsErr: errStorage}
	svc := &Service{opts: Options{storage: store}}
	admin := structs.User{ID: config.UserID("admin@example.com"), Username: "admin", Role: config.RoleAdmin}

	got, hasNext, err := svc.ListAdminRequests(context.Background(), admin, 2, 10)

	require.Nil(t, got)
	require.False(t, hasNext)
	require.ErrorIs(t, err, errStorage)
	require.Equal(t, []int64{11}, store.listRequestsLimitsSet())
	require.Equal(t, []int64{10}, store.listRequestsOffsetsSet())
}
