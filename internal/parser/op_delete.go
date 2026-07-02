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

package parser

import (
	"fmt"
	"slices"

	"github.com/kazhuravlev/just"
	pg "github.com/pganalyze/pg_query_go/v6"
)

type DeleteVec struct { //nolint:recvcheck
	Tbl       string
	Target    []string
	Filter    []string
	Returning []string
}

func (s *DeleteVec) Columns() []string {
	columns := just.SliceUniq(slices.Concat(s.Target, s.Filter, s.Returning))

	return columns
}

func (DeleteVec) isVector() {}

func handleDelete(req *pg.DeleteStmt) ([]Vector, error) {
	if req.UsingClause != nil ||
		req.GetWithClause() != nil {
		return nil, fmt.Errorf("unknown clause: %w", ErrNotImplemented)
	}

	tables, fqTableName, err := deleteTables(req)
	if err != nil {
		return nil, err
	}

	retCols, err := pNodes2Columns(req.GetReturningList(), fqTableName)
	if err != nil {
		return nil, fmt.Errorf("parse returning: %w", err)
	}

	whereColumns, err := parseWhereClause(req.GetWhereClause())
	if err != nil {
		return nil, fmt.Errorf("parse where: %w", err)
	}

	allTables, err := tables.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all tables: %w", err)
	}

	table2filter, err := columnsByTable(tables, allTables, whereColumns)
	if err != nil {
		return nil, err
	}
	table2returning, err := columnsByTable(tables, allTables, retCols)
	if err != nil {
		return nil, err
	}

	vectors := make([]Vector, 0, len(allTables))
	for _, tbl := range allTables {
		vectors = append(vectors, DeleteVec{
			Tbl:       tbl,
			Target:    nil,
			Filter:    table2filter[tbl].ListNames(),
			Returning: table2returning[tbl].ListNames(),
		})
	}

	return vectors, nil
}

func deleteTables(req *pg.DeleteStmt) (*Tables, string, error) {
	tables := NewTables("public")
	rel := req.GetRelation()
	fqTableName, err := tables.Put(rel.GetCatalogname(), rel.GetSchemaname(), rel.GetRelname(), rel.GetAlias().GetAliasname())
	if err != nil {
		return nil, "", fmt.Errorf("failed to add table: %w", err)
	}
	if err := tables.Finalize(); err != nil {
		return nil, "", fmt.Errorf("failed to finalize tables: %w", err)
	}

	return tables, fqTableName, nil
}
