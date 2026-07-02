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

package storage //nolint:testpackage

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/storage/jetgen/model"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
	"github.com/stretchr/testify/require"
)

func TestAdaptEmptyListsReturnsNonNilSlices(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		check func(t *testing.T)
	}{
		{
			name: "query results",
			check: func(t *testing.T) {
				t.Helper()

				var items []model.QueryResults

				got := adaptQueryResults(items)
				require.NotNil(t, got)
				require.Empty(t, got)
			},
		},
		{
			name: "bookmarks",
			check: func(t *testing.T) {
				t.Helper()

				var items []model.Bookmarks

				got := adaptBookmarks(items)
				require.NotNil(t, got)
				require.Empty(t, got)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.check(t)
		})
	}
}

func TestAdaptQueryResultsMapsNonEmptyRows(t *testing.T) {
	t.Parallel()

	id := uuid6.FromStr("01J2Z7K2YJ9G8F7E6D5C4B3A2H")
	createdAt := time.Date(2026, time.July, 2, 10, 11, 12, 13, time.UTC)
	response := []byte(`{"columns":["id"],"rows":[["1"]]}`)

	got := adaptQueryResults([]model.QueryResults{
		{
			ID:        id,
			UserID:    config.UserID("user-1"),
			TargetID:  config.TargetID("target-1"),
			CreatedAt: createdAt,
			Query:     "select 1",
			State:     structs.QueryStateCompleted,
			Response:  response,
		},
	})

	require.Equal(t, []QueryResult{
		{
			ID:        id,
			UserID:    config.UserID("user-1"),
			TargetID:  config.TargetID("target-1"),
			CreatedAt: createdAt,
			Query:     "select 1",
			State:     structs.QueryStateCompleted,
			Response:  response,
		},
	}, got)
}

func TestAdaptBookmarksMapsNonEmptyRows(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("018aacd7-8b53-75f2-a46c-8a87f37a7b2f")
	createdAt := time.Date(2026, time.July, 2, 13, 14, 15, 16, time.UTC)

	got := adaptBookmarks([]model.Bookmarks{
		{
			ID:        id,
			UserID:    "user-1",
			TargetID:  "target-1",
			Title:     "Important query",
			Query:     "select * from users",
			CreatedAt: createdAt,
		},
	})

	require.Equal(t, []Bookmark{
		{
			ID:        uuid6.FromUUID(id),
			UserID:    config.UserID("user-1"),
			TargetID:  config.TargetID("target-1"),
			Title:     "Important query",
			Query:     "select * from users",
			CreatedAt: createdAt,
		},
	}, got)
}
