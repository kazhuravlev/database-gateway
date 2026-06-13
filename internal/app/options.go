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

package app

import (
	"context"
	"log/slog"

	"github.com/go-jet/jet/v2/qrm"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/policy"
	"github.com/kazhuravlev/database-gateway/internal/storage"
	"github.com/kazhuravlev/database-gateway/internal/storage/jetgen/model"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
)

type metadataStore interface {
	Conn(ctx context.Context) qrm.DB
	InsertQueryResults(conn qrm.DB, req storage.InsertQueryResultsReq) error
	SetQueryResultsState(conn qrm.DB, req storage.SetQueryResultsStateReq) error
	GetQueryResultsByID(conn qrm.DB, queryID uuid6.UUID) (*model.QueryResults, error)
	ListQueryResultsByUser(conn qrm.DB, uid config.UserID, limit int64) ([]storage.QueryResult, error)
	ListQueryResults(conn qrm.DB, limit, offset int64) ([]storage.QueryResult, error)
	InsertBookmark(conn qrm.DB, req storage.InsertBookmarkReq) error
	DeleteBookmark(conn qrm.DB, uid config.UserID, bookmarkID uuid6.UUID) error
	ListBookmarks(conn qrm.DB, uid config.UserID, targetID config.TargetID) ([]storage.Bookmark, error)
	ListBookmarksByUser(conn qrm.DB, uid config.UserID) ([]storage.Bookmark, error)
}

//go:generate toolset run options-gen -from-struct=Options
type Options struct {
	logger     *slog.Logger             `option:"mandatory" validate:"required"`
	targets    []config.Target          `option:"mandatory" validate:"required"`
	users      config.UsersProviderOIDC `option:"mandatory" validate:"required"`
	authorizer policy.Authorizer        `option:"mandatory" validate:"required"`
	storage    metadataStore            `option:"mandatory" validate:"required"`
}
