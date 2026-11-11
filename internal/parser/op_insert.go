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

type InsertVec struct { //nolint:recvcheck
	Tbl       string
	Target    []string
	Returning []string
	Read      []string
}

func (s *InsertVec) Columns() []string {
	columns := just.SliceUniq(append(append([]string{}, s.Target...), s.Returning...))

	return columns
}

func (InsertVec) isVector() {}

func handleInsert(req *pg.InsertStmt) ([]Vector, error) { //nolint:gocyclo,gocognit,cyclop,funlen
	if req.GetWithClause() != nil ||
		req.GetOverride() != pg.OverridingKind_OVERRIDING_NOT_SET {
		return nil, fmt.Errorf("unknown clause: %w", ErrNotImplemented)
	}

	rel := req.GetRelation()
	tables := NewTables("public")
	_, err := tables.Put(rel.GetCatalogname(), rel.GetSchemaname(), rel.GetRelname(), rel.GetAlias().GetAliasname())
	if err != nil {
		return nil, fmt.Errorf("failed to add table: %w", err)
	}
	if err := tables.Finalize(); err != nil {
		return nil, fmt.Errorf("failed to finalize tables: %w", err)
	}

	targetCols, err := parseInsertColumns(req.GetCols())
	if err != nil {
		return nil, fmt.Errorf("parse target columns: %w", err)
	}

	switch node := req.GetSelectStmt().GetNode().(type) {
	default:
		return nil, fmt.Errorf("target select type (%T): %w", node, ErrNotImplemented)
	case *pg.Node_SelectStmt:
		sel := node.SelectStmt

		if sel.DistinctClause != nil ||
			sel.GetIntoClause() != nil ||
			sel.TargetList != nil ||
			sel.FromClause != nil ||
			sel.GetWhereClause() != nil ||
			sel.GroupClause != nil ||
			sel.GetGroupDistinct() ||
			sel.GetHavingClause() != nil ||
			sel.WindowClause != nil ||
			sel.SortClause != nil ||
			sel.GetLimitOffset() != nil ||
			sel.GetLimitCount() != nil ||
			sel.GetLimitOption() != pg.LimitOption_LIMIT_OPTION_DEFAULT ||
			sel.LockingClause != nil ||
			sel.GetWithClause() != nil ||
			sel.GetOp() != pg.SetOperation_SETOP_NONE ||
			sel.GetAll() ||
			sel.GetLarg() != nil ||
			sel.GetRarg() != nil {
			return nil, fmt.Errorf("unknown clause: %w", ErrNotImplemented)
		}

		if len(sel.GetFromClause()) != 0 {
			return nil, fmt.Errorf("insert from select: %w", ErrNotImplemented)
		}

		for _, node := range sel.GetValuesLists() {
			switch node := node.GetNode().(type) {
			default:
				return nil, fmt.Errorf("unknown node type (%T): %w", node, ErrNotImplemented)
			case *pg.Node_List:
				for _, node := range node.List.GetItems() {
					switch node := node.GetNode().(type) {
					default:
						return nil, fmt.Errorf("unknown node list node type (%T): %w", node, ErrNotImplemented)
					// Allow `insert ... values (DEFAULT)`
					case *pg.Node_SetToDefault:
					// Allow `insert ... values (1)`
					// Allow `insert ... values ('1')`
					case *pg.Node_AConst:
					}
				}
			}
		}
	case nil: // This is `insert into t1 default values`
	}

	retCols, err := parseReturning(req.GetReturningList())
	if err != nil {
		return nil, fmt.Errorf("parse returning columns: %w", err)
	}

	var conflictUpdate bool
	var updateCols, readCols Columns
	if confl := req.GetOnConflictClause(); confl != nil {
		switch confl.GetAction() { //nolint:exhaustive
		default:
			return nil, fmt.Errorf("unknonw action on conflict: %w", ErrNotImplemented)
		case pg.OnConflictAction_ONCONFLICT_NONE, pg.OnConflictAction_ONCONFLICT_NOTHING, pg.OnConflictAction_ONCONFLICT_UPDATE:
		}

		if confl.GetWhereClause() != nil ||
			confl.GetInfer().GetWhereClause() != nil ||
			confl.GetInfer().GetConname() != "" {
			return nil, fmt.Errorf("unknown on-conflict clause: %w", ErrNotImplemented)
		}

		conflictCols, err := parseConflictColumns(confl.GetInfer().GetIndexElems())
		if err != nil {
			return nil, fmt.Errorf("parse conflict columns: %w", err)
		}

		conflictTargetCols, conflictReads, err := parseAssignments(confl.GetTargetList())
		if err != nil {
			return nil, fmt.Errorf("parse target columns: %w", err)
		}

		targetCols = append(targetCols, conflictCols...)
		targetCols = append(targetCols, conflictTargetCols...)
		conflictUpdate = confl.GetAction() == pg.OnConflictAction_ONCONFLICT_UPDATE
		updateCols = conflictTargetCols
		readCols = slices.Concat(conflictCols, conflictReads)
	}

	allTables, err := tables.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all tables: %w", err)
	}

	table2target, err := columnsByTable(tables, allTables, targetCols)
	if err != nil {
		return nil, err
	}
	table2returning, err := columnsByTable(tables, allTables, retCols)
	if err != nil {
		return nil, err
	}

	var readNames []string
	for _, col := range readCols {
		readNames = append(readNames, col.column)
	}
	vectors := make([]Vector, 0, len(allTables))
	for _, tbl := range allTables {
		vectors = append(vectors, InsertVec{
			Tbl:       tbl,
			Target:    table2target[tbl].ListNames(),
			Returning: table2returning[tbl].ListNames(),
			Read:      readNames,
		})
	}

	if conflictUpdate {
		vectors = append(vectors, UpdateVec{Tbl: allTables[0], Target: updateCols.ListNames(), Filter: nil, Returning: nil, Read: nil})
	}

	return vectors, nil
}
