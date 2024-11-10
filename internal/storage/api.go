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
	CreatedAt time.Time
	Query     string
	Response  json.RawMessage
}

func (s *Service) InsertQueryResults(conn qrm.DB, req InsertQueryResultsReq) error {
	obj := model.QueryResults{
		ID:        req.ID,
		UserID:    req.UserID,
		CreatedAt: req.CreatedAt,
		Query:     req.Query,
		Response:  req.Response,
	}
	res, err := tbl.QueryResults.
		INSERT(tbl.QueryResults.AllColumns).
		MODEL(obj).
		Exec(conn)
	if err := handleError("insert query results", err, res); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetQueryResults(conn qrm.DB, uid config.UserID, id uuid6.UUID) (*model.QueryResults, error) {
	var obj model.QueryResults
	q := tbl.QueryResults.
		SELECT(tbl.QueryResults.AllColumns).
		WHERE(postgres.AND(
			tbl.QueryResults.ID.EQ(postgres.UUID(id.ToUUID())),
			tbl.QueryResults.UserID.EQ(postgres.String(uid.S())),
		)).
		LIMIT(1)
	err := q.Query(conn, &obj)
	if err := handleError("get query results", err, nil); err != nil {
		return nil, err
	}

	return &obj, nil
}
