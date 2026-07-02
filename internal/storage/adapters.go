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

package storage

import (
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/storage/jetgen/model"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
)

func adaptQueryResults(items []model.QueryResults) []QueryResult {
	out := make([]QueryResult, 0, len(items))
	for i := range items {
		out = append(out, adaptQueryResult(&items[i]))
	}

	return out
}

func adaptQueryResult(item *model.QueryResults) QueryResult {
	return QueryResult{
		ID:        item.ID,
		UserID:    item.UserID,
		TargetID:  item.TargetID,
		CreatedAt: item.CreatedAt,
		Query:     item.Query,
		State:     item.State,
		Response:  item.Response,
	}
}

func adaptBookmarks(items []model.Bookmarks) []Bookmark {
	out := make([]Bookmark, 0, len(items))
	for i := range items {
		out = append(out, adaptBookmark(&items[i]))
	}

	return out
}

func adaptBookmark(item *model.Bookmarks) Bookmark {
	return Bookmark{
		ID:        uuid6.FromUUID(item.ID),
		UserID:    config.UserID(item.UserID),
		TargetID:  config.TargetID(item.TargetID),
		Title:     item.Title,
		Query:     item.Query,
		CreatedAt: item.CreatedAt,
	}
}
