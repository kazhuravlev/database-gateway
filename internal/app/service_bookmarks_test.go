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
	"errors"
	"sync"
	"testing"

	"github.com/go-jet/jet/v2/qrm"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/storage"
	"github.com/kazhuravlev/database-gateway/internal/storage/jetgen/model"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
	"github.com/stretchr/testify/require"
)

var errBookmarkStorageFailed = errors.New("storage failed")

type fakeBookmarkStorage struct {
	mu              sync.Mutex
	bookmarkInserts []storage.InsertBookmarkReq
	insertErr       error
}

func (*fakeBookmarkStorage) Conn(context.Context) qrm.DB { //nolint:ireturn
	return nil
}

func (*fakeBookmarkStorage) InsertQueryResults(qrm.DB, storage.InsertQueryResultsReq) error {
	return nil
}

func (*fakeBookmarkStorage) SetQueryResultsState(qrm.DB, storage.SetQueryResultsStateReq) error {
	return nil
}

func (*fakeBookmarkStorage) GetQueryResultsByID(qrm.DB, uuid6.UUID) (*model.QueryResults, error) {
	return nil, storage.ErrNotFound
}

func (*fakeBookmarkStorage) ListQueryResultsByUser(
	qrm.DB,
	config.UserID,
	int64,
) ([]storage.QueryResult, error) {
	return nil, nil
}

func (*fakeBookmarkStorage) ListQueryResults(qrm.DB, int64, int64) ([]storage.QueryResult, error) {
	return nil, nil
}

func (s *fakeBookmarkStorage) InsertBookmark(_ qrm.DB, req storage.InsertBookmarkReq) error { //nolint:gocritic
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bookmarkInserts = append(s.bookmarkInserts, req)
	if s.insertErr != nil {
		return s.insertErr
	}

	return nil
}

func (*fakeBookmarkStorage) DeleteBookmark(qrm.DB, config.UserID, uuid6.UUID) error {
	return nil
}

func (*fakeBookmarkStorage) ListBookmarks(qrm.DB, config.UserID, config.TargetID) ([]storage.Bookmark, error) {
	return nil, nil
}

func (*fakeBookmarkStorage) ListBookmarksByUser(qrm.DB, config.UserID) ([]storage.Bookmark, error) {
	return nil, nil
}

func (s *fakeBookmarkStorage) insertedBookmarks() []storage.InsertBookmarkReq {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]storage.InsertBookmarkReq(nil), s.bookmarkInserts...)
}

func TestServiceAddBookmarkTrimsTitleAndQuery(t *testing.T) {
	t.Parallel()

	store := new(fakeBookmarkStorage)
	svc := newBookmarkTestService(t, store, bookmarkAllowPolicy)
	user := structs.User{ID: config.UserID("alice@example.com"), Username: "alice", Role: config.RoleUser}

	err := svc.AddBookmark(context.Background(), user, config.TargetID("pg-1"), "  Important query  ", "\n select 1 \t")

	require.NoError(t, err)
	inserted := store.insertedBookmarks()
	require.Len(t, inserted, 1)
	require.NotEmpty(t, inserted[0].ID.S())
	require.Equal(t, user.ID, inserted[0].UserID)
	require.Equal(t, config.TargetID("pg-1"), inserted[0].TargetID)
	require.Equal(t, "Important query", inserted[0].Title)
	require.Equal(t, "select 1", inserted[0].Query)
	require.False(t, inserted[0].CreatedAt.IsZero())
}

func TestServiceAddBookmarkValidatesEmptyTitleAndQuery(t *testing.T) {
	t.Parallel()

	user := structs.User{ID: config.UserID("alice@example.com"), Username: "alice", Role: config.RoleUser}
	testCases := []struct {
		name  string
		title string
		query string
	}{
		{name: "empty title", title: "", query: "select 1"},
		{name: "blank title", title: " \t", query: "select 1"},
		{name: "empty query", title: "Important query", query: ""},
		{name: "blank query", title: "Important query", query: " \n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := new(fakeBookmarkStorage)
			svc := newBookmarkTestService(t, store, bookmarkAllowPolicy)

			err := svc.AddBookmark(context.Background(), user, config.TargetID("pg-1"), tc.title, tc.query)

			require.ErrorContains(t, err, "title and query are required")
			require.Empty(t, store.insertedBookmarks())
		})
	}
}

func TestServiceAddBookmarkValidatesTargetAccess(t *testing.T) {
	t.Parallel()

	store := new(fakeBookmarkStorage)
	svc := newBookmarkTestService(t, store, `
package gateway

default allow_target := false
default allow_query := false
`)
	user := structs.User{ID: config.UserID("alice@example.com"), Username: "alice", Role: config.RoleUser}

	err := svc.AddBookmark(context.Background(), user, config.TargetID("pg-1"), "Important query", "select 1")

	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorContains(t, err, "validate target access")
	require.Empty(t, store.insertedBookmarks())
}

func TestServiceAddBookmarkPropagatesStorageError(t *testing.T) {
	t.Parallel()

	insertErr := errBookmarkStorageFailed
	store := &fakeBookmarkStorage{insertErr: insertErr}
	svc := newBookmarkTestService(t, store, bookmarkAllowPolicy)
	user := structs.User{ID: config.UserID("alice@example.com"), Username: "alice", Role: config.RoleUser}

	err := svc.AddBookmark(context.Background(), user, config.TargetID("pg-1"), "Important query", "select 1")

	require.ErrorIs(t, err, insertErr)
	require.ErrorContains(t, err, "insert bookmark")
	require.Len(t, store.insertedBookmarks(), 1)
}

const bookmarkAllowPolicy = `
package gateway

default allow_target := false
default allow_query := false

allow_target if {
	"role:user" in input.subjects
	input.target == "pg-1"
}
`

func newBookmarkTestService(t *testing.T, store *fakeBookmarkStorage, policy string) *Service {
	t.Helper()

	return &Service{
		opts: Options{
			targets: []config.Target{
				{
					ID:            "pg-1",
					Description:   "main",
					Tags:          []string{"prod"},
					Type:          "postgres",
					DefaultSchema: "public",
					Tables:        []config.TargetTable{{Table: "public.clients", Fields: []string{"id"}}},
				},
			},
			authorizer: mustAuthorizer(t, policy),
			storage:    store,
		},
		connsMu: new(sync.RWMutex),
		conns:   nil,
	}
}
