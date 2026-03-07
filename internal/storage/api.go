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
	"encoding/json"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/storage/jetgen/model"
	tbl "github.com/kazhuravlev/database-gateway/internal/storage/jetgen/table"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
)

type InsertQueryResultsReq struct {
	ID        uuid6.UUID
	UserID    config.UserID
	TargetID  config.TargetID
	CreatedAt time.Time
	Query     string
	Response  json.RawMessage
}

func (*Service) InsertQueryResults(conn qrm.DB, req InsertQueryResultsReq) error { //nolint:gocritic
	obj := model.QueryResults{
		ID:        req.ID,
		UserID:    req.UserID,
		TargetID:  req.TargetID,
		CreatedAt: req.CreatedAt,
		Query:     req.Query,
		Response:  req.Response,
	}
	//nolint:unqueryvet // ok while reading into model
	res, err := tbl.QueryResults.
		INSERT(tbl.QueryResults.AllColumns).
		MODEL(obj).
		Exec(conn)
	if err := handleError("insert query results", err, res); err != nil {
		return err
	}

	return nil
}

func (*Service) GetQueryResults(conn qrm.DB, uid config.UserID, queryID uuid6.UUID) (*model.QueryResults, error) {
	var obj model.QueryResults
	//nolint:unqueryvet // ok while reading into model
	err := tbl.QueryResults.
		SELECT(tbl.QueryResults.AllColumns).
		WHERE(postgres.AND(
			tbl.QueryResults.ID.EQ(postgres.UUID(queryID.ToUUID())),
			tbl.QueryResults.UserID.EQ(postgres.String(uid.S())),
		)).
		LIMIT(1).
		Query(conn, &obj)
	if err := handleError("get query results", err, nil); err != nil {
		return nil, err
	}

	return &obj, nil
}

func (*Service) ListQueryResultsByUser(conn qrm.DB, uid config.UserID, limit int64) ([]QueryResult, error) {
	var items []model.QueryResults
	//nolint:unqueryvet // ok while reading into model
	err := tbl.QueryResults.
		SELECT(tbl.QueryResults.AllColumns).
		WHERE(tbl.QueryResults.UserID.EQ(postgres.String(uid.S()))).
		ORDER_BY(tbl.QueryResults.CreatedAt.DESC()).
		LIMIT(limit).
		Query(conn, &items)
	if err := handleError("list query results by user", err, nil); err != nil {
		return nil, err
	}

	if items == nil {
		return []QueryResult{}, nil
	}

	out := make([]QueryResult, 0, len(items))
	for _, item := range items {
		out = append(out, QueryResult{
			ID:        item.ID,
			UserID:    item.UserID,
			TargetID:  item.TargetID,
			CreatedAt: item.CreatedAt,
			Query:     item.Query,
			Response:  item.Response,
		})
	}

	return out, nil
}

type InsertBookmarkReq struct {
	ID        uuid6.UUID
	UserID    config.UserID
	TargetID  config.TargetID
	Title     string
	Query     string
	CreatedAt time.Time
}

func (*Service) InsertBookmark(conn qrm.DB, req InsertBookmarkReq) error { //nolint:gocritic
	obj := model.Bookmarks{
		ID:        req.ID.ToUUID(),
		UserID:    req.UserID.S(),
		TargetID:  req.TargetID.S(),
		Title:     req.Title,
		Query:     req.Query,
		CreatedAt: req.CreatedAt,
	}
	//nolint:unqueryvet // ok while reading into model
	res, err := tbl.Bookmarks.
		INSERT(tbl.Bookmarks.AllColumns).
		MODEL(obj).
		Exec(conn)
	if err := handleError("insert bookmark", err, res); err != nil {
		return err
	}

	return nil
}

func (*Service) DeleteBookmark(conn qrm.DB, uid config.UserID, bookmarkID uuid6.UUID) error {
	//nolint:unqueryvet // ok while reading into model
	res, err := tbl.Bookmarks.
		DELETE().
		WHERE(postgres.AND(
			tbl.Bookmarks.ID.EQ(postgres.UUID(bookmarkID.ToUUID())),
			tbl.Bookmarks.UserID.EQ(postgres.String(uid.S())),
		)).
		Exec(conn)
	if err := handleError("delete bookmark", err, res); err != nil {
		return err
	}

	return nil
}

func (*Service) ListBookmarks(conn qrm.DB, uid config.UserID, targetID config.TargetID) ([]Bookmark, error) {
	var items []model.Bookmarks
	//nolint:unqueryvet // ok while reading into model
	err := tbl.Bookmarks.
		SELECT(tbl.Bookmarks.AllColumns).
		WHERE(postgres.AND(
			tbl.Bookmarks.UserID.EQ(postgres.String(uid.S())),
			tbl.Bookmarks.TargetID.EQ(postgres.String(targetID.S())),
		)).
		ORDER_BY(tbl.Bookmarks.CreatedAt.DESC()).
		Query(conn, &items)
	if err := handleError("list bookmarks", err, nil); err != nil {
		return nil, err
	}

	if items == nil {
		return []Bookmark{}, nil
	}

	out := make([]Bookmark, 0, len(items))
	for _, item := range items {
		out = append(out, Bookmark{
			ID:        uuid6.FromUUID(item.ID),
			UserID:    config.UserID(item.UserID),
			TargetID:  config.TargetID(item.TargetID),
			Title:     item.Title,
			Query:     item.Query,
			CreatedAt: item.CreatedAt,
		})
	}

	return out, nil
}

func (*Service) ListBookmarksByUser(conn qrm.DB, uid config.UserID) ([]Bookmark, error) {
	var items []model.Bookmarks
	//nolint:unqueryvet // ok while reading into model
	err := tbl.Bookmarks.
		SELECT(tbl.Bookmarks.AllColumns).
		WHERE(tbl.Bookmarks.UserID.EQ(postgres.String(uid.S()))).
		ORDER_BY(tbl.Bookmarks.CreatedAt.DESC()).
		Query(conn, &items)
	if err := handleError("list bookmarks by user", err, nil); err != nil {
		return nil, err
	}

	if items == nil {
		return []Bookmark{}, nil
	}

	out := make([]Bookmark, 0, len(items))
	for _, item := range items {
		out = append(out, Bookmark{
			ID:        uuid6.FromUUID(item.ID),
			UserID:    config.UserID(item.UserID),
			TargetID:  config.TargetID(item.TargetID),
			Title:     item.Title,
			Query:     item.Query,
			CreatedAt: item.CreatedAt,
		})
	}

	return out, nil
}
