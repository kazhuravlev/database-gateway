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
	Tbl    string
	Target []string
}

func (s *InsertVec) Columns() []string {
	columns := just.SliceUniq(s.Target)

	return columns
}

func (InsertVec) isVector() {}

func handleInsert(req *pg.InsertStmt) ([]Vector, error) { //nolint:gocyclo,gocognit,cyclop,funlen
	if req.GetWithClause() != nil ||
		req.GetOverride() != pg.OverridingKind_OVERRIDING_NOT_SET {
		return nil, fmt.Errorf("unknown clause: %w", ErrNotImplemented)
	}

	rel := req.GetRelation()
	tables := Tables{m: make(map[string]string)}
	fqTableName := tables.Put(rel.GetCatalogname(), rel.GetSchemaname(), rel.GetRelname(), rel.GetAlias().GetAliasname())
	tables.Finalize()

	targetCols, err := pNodes2Columns(req.GetCols(), fqTableName)
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

	retCols, err := pNodes2Columns(req.GetReturningList(), fqTableName)
	if err != nil {
		return nil, fmt.Errorf("parse returning columns: %w", err)
	}

	allColumns := slices.Concat(targetCols, retCols)

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

		conflictCols, err := pNodes2Columns(confl.GetInfer().GetIndexElems(), fqTableName)
		if err != nil {
			return nil, fmt.Errorf("parse conflict columns: %w", err)
		}

		targetCols, err := pNodes2Columns(confl.GetTargetList(), fqTableName)
		if err != nil {
			return nil, fmt.Errorf("parse target columns: %w", err)
		}

		allColumns = slices.Concat(allColumns, conflictCols, targetCols)
	}

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
		vectors = append(vectors, InsertVec{
			Tbl:    tbl,
			Target: cols.ListNames(),
		})
	}

	return vectors, nil
}

func pNodes2Columns(nodes []*pg.Node, defaultTable string) (Columns, error) { //nolint:cyclop
	var allColumns Columns

	for _, ret := range nodes {
		switch node := ret.GetNode().(type) {
		default:
			return nil, fmt.Errorf("col type (%T): %w", ret.GetNode(), ErrNotImplemented)
		case *pg.Node_IndexElem:
			idx := node.IndexElem
			if idx.GetExpr() != nil ||
				idx.GetIndexcolname() != "" ||
				idx.Collation != nil ||
				idx.Opclass != nil ||
				idx.Opclassopts != nil ||
				idx.GetOrdering() != pg.SortByDir_SORTBY_DEFAULT ||
				idx.GetNullsOrdering() != pg.SortByNulls_SORTBY_NULLS_DEFAULT {
				return nil, fmt.Errorf("unknown index elem: %w", ErrNotImplemented)
			}

			col := Column{
				table:  []string{defaultTable},
				column: idx.GetName(),
			}
			allColumns = append(allColumns, col)
		case *pg.Node_ResTarget:
			resTarget := node.ResTarget
			if resTarget.Indirection != nil {
				return nil, fmt.Errorf("resTarget field: %w", ErrNotImplemented)
			}

			if resTarget.GetName() != "" {
				col := Column{
					table:  []string{defaultTable},
					column: resTarget.GetName(),
				}
				allColumns = append(allColumns, col)

				continue
			}

			switch node := resTarget.GetVal().GetNode().(type) {
			default:
				return nil, fmt.Errorf("resTarget type (%T): %w", resTarget.GetVal().GetNode(), ErrNotImplemented)
			case *pg.Node_ColumnRef:
				column, err := pNodeColumnRef(node)
				if err != nil {
					return nil, fmt.Errorf("parse column: %w", err)
				}

				allColumns = append(allColumns, column)
			}
		}
	}

	return allColumns, nil
}
