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
	Tbl    string
	Target []string
	Filter []string
}

func (s *DeleteVec) Columns() []string {
	columns := just.SliceUniq(slices.Concat(s.Target, s.Filter))

	return columns
}

func (DeleteVec) isVector() {}

func handleDelete(req *pg.DeleteStmt) ([]Vector, error) { //nolint:gocyclo
	if req.UsingClause != nil ||
		req.GetWithClause() != nil {
		return nil, fmt.Errorf("unknown clause: %w", ErrNotImplemented)
	}

	tables := NewTables("public")
	rel := req.GetRelation()
	fqTableName, err := tables.Put(rel.GetCatalogname(), rel.GetSchemaname(), rel.GetRelname(), rel.GetAlias().GetAliasname())
	if err != nil {
		return nil, fmt.Errorf("failed to add table: %w", err)
	}
	if err := tables.Finalize(); err != nil {
		return nil, fmt.Errorf("failed to finalize tables: %w", err)
	}

	retCols, err := pNodes2Columns(req.GetReturningList(), fqTableName)
	if err != nil {
		return nil, fmt.Errorf("parse returning: %w", err)
	}

	whereColumns, err := parseUpdateWhere(req.GetWhereClause())
	if err != nil {
		return nil, fmt.Errorf("parse where: %w", err)
	}

	allColumns := slices.Concat(retCols, whereColumns)

	table2target := make(map[string]Columns, len(allColumns))
	for _, column := range allColumns {
		tbl, ok := tables.Get(column.Table())
		if !ok {
			return nil, fmt.Errorf("table not found: %s", column.Table()) //nolint:err113
		}

		table2target[tbl] = append(table2target[tbl], column)
	}

	vectors := make([]Vector, 0, len(table2target))
	for tbl, cols := range table2target {
		vectors = append(vectors, DeleteVec{
			Tbl:    tbl,
			Target: cols.ListNames(),
			Filter: nil, // TODO: impl
		})
	}

	return vectors, nil
}
