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
	"errors"
	"fmt"
	"slices"

	"github.com/kazhuravlev/just"
	pg "github.com/pganalyze/pg_query_go/v6"
)

type UpdateVec struct { //nolint:recvcheck
	Tbl    string
	Target []string
	Filter []string
}

func (s *UpdateVec) Columns() []string {
	columns := just.SliceUniq(slices.Concat(s.Target, s.Filter))

	return columns
}

func (UpdateVec) isVector() {}

func parseAexpr(node *pg.Node_AExpr) (Columns, error) { //nolint:cyclop
	var columns Columns

	expr := node.AExpr

	switch expr.GetKind() { //nolint:exhaustive // this is whitelist of allowed kinds
	default:
		return nil, fmt.Errorf("aexpr clause kind (%s): %w", expr.GetKind().String(), ErrNotImplemented)
	case pg.A_Expr_Kind_AEXPR_OP, pg.A_Expr_Kind_AEXPR_IN, pg.A_Expr_Kind_AEXPR_LIKE, pg.A_Expr_Kind_AEXPR_ILIKE, pg.A_Expr_Kind_AEXPR_BETWEEN:
		if len(expr.GetName()) != 1 {
			return nil, errors.New("clause aexpr name len should be 1") //nolint:err113
		}

		switch expr := expr.GetName()[0].GetNode().(type) {
		default:
			return nil, fmt.Errorf("aexpr clause name (%T): %w", expr, ErrNotImplemented)
		case *pg.Node_String_:
			switch expr.String_.GetSval() {
			default:
				return nil, fmt.Errorf("aexpr clause operator (%s): %w", expr.String_.GetSval(), ErrNotImplemented)
			// NOTE: allowed operators
			case "*", "-", "+", "=", ">", "<", "<=", ">=", "!=", "<>", "~~", "||", "BETWEEN":
			}
		}

		switch left := expr.GetLexpr().GetNode().(type) {
		default:
			return nil, fmt.Errorf("aexpr clause left expr (%T): %w", left, ErrNotImplemented)
		case *pg.Node_AExpr:
		case *pg.Node_ColumnRef:
			column, err := pNodeColumnRef(left)
			if err != nil {
				return nil, fmt.Errorf("parse column: %w", err)
			}

			columns = append(columns, column)
		}

		switch right := expr.GetRexpr().GetNode().(type) {
		default:
			return nil, fmt.Errorf("aexpr clause right expr (%T): %w", right, ErrNotImplemented)
		case *pg.Node_AConst:
		case *pg.Node_ColumnRef:
			column, err := pNodeColumnRef(right)
			if err != nil {
				return nil, fmt.Errorf("parse column: %w", err)
			}

			columns = append(columns, column)
		case *pg.Node_List:
			for _, node := range right.List.GetItems() {
				switch node := node.GetNode().(type) {
				default:
					return nil, fmt.Errorf("node item in aexpr filter (%T): %w", node, ErrNotImplemented)
				case *pg.Node_AConst:
				}
			}
		}
	}

	return columns, nil
}

func handleUpdate(req *pg.UpdateStmt) ([]Vector, error) { //nolint:gocyclo,cyclop,funlen,gocognit
	if req.FromClause != nil ||
		req.GetWithClause() != nil {
		return nil, fmt.Errorf("unknown clause: %w", ErrNotImplemented)
	}

	rel := req.GetRelation()
	tables := NewTables("public")
	fqTableName, err := tables.Put(rel.GetCatalogname(), rel.GetSchemaname(), rel.GetRelname(), rel.GetAlias().GetAliasname())
	if err != nil {
		return nil, fmt.Errorf("failed to add table: %w", err)
	}
	if err := tables.Finalize(); err != nil {
		return nil, fmt.Errorf("failed to finalize tables: %w", err)
	}

	var targetCols Columns
	for _, node := range req.GetTargetList() {
		switch node := node.GetNode().(type) {
		default:
			return nil, fmt.Errorf("col type (%T): %w", node, ErrNotImplemented)
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
				table:  []string{fqTableName},
				column: idx.GetName(),
			}
			targetCols = append(targetCols, col)
		case *pg.Node_ResTarget:
			resTarget := node.ResTarget
			if resTarget.Indirection != nil {
				return nil, fmt.Errorf("resTarget field: %w", ErrNotImplemented)
			}

			// Column name
			targetCols = append(targetCols, Column{
				table:  []string{fqTableName},
				column: resTarget.GetName(),
			})

			// Column value
			switch node := resTarget.GetVal().GetNode().(type) {
			default:
				return nil, fmt.Errorf("resTarget type (%T): %w", node, ErrNotImplemented)
			case *pg.Node_AConst:
			case *pg.Node_SetToDefault:
			case *pg.Node_AExpr:
				cols, err := parseAexpr(node)
				if err != nil {
					return nil, fmt.Errorf("resTarget expr: %w", err)
				}

				targetCols = append(targetCols, cols...)
			case *pg.Node_ColumnRef:
				column, err := pNodeColumnRef(node)
				if err != nil {
					return nil, fmt.Errorf("parse column: %w", err)
				}

				targetCols = append(targetCols, column)
			}
		}
	}

	retCols, err := pNodes2Columns(req.GetReturningList(), fqTableName)
	if err != nil {
		return nil, fmt.Errorf("parse returning columns: %w", err)
	}

	whereColumns, err := parseUpdateWhere(req.GetWhereClause())
	if err != nil {
		return nil, fmt.Errorf("parse where clause: %w", err)
	}

	allColumns := slices.Concat(targetCols, retCols, whereColumns)

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
		vectors = append(vectors, UpdateVec{
			Tbl:    tbl,
			Target: cols.ListNames(),
			Filter: nil, // TODO: impl
		})
	}

	return vectors, nil
}

func parseUpdateWhere(node *pg.Node) (Columns, error) { //nolint:cyclop
	var columns Columns

	switch node := node.GetNode().(type) {
	default:
		return nil, fmt.Errorf("where clause not supported (%T): %w", node, ErrNotImplemented)
	case nil, *pg.Node_NullTest:
	case *pg.Node_BoolExpr:
		for _, node := range node.BoolExpr.GetArgs() {
			switch node.GetNode().(type) {
			default:
				return nil, fmt.Errorf("bool expr argument (%T): %w", node.GetNode(), ErrNotImplemented)
			case *pg.Node_BoolExpr:
				cols, err := parseUpdateWhere(node)
				if err != nil {
					return nil, fmt.Errorf("parse update where bool: %w", err)
				}
				columns = append(columns, cols...)
			case *pg.Node_AExpr:
				cols, err := parseUpdateWhere(node)
				if err != nil {
					return nil, fmt.Errorf("parse update where bool 2: %w", err)
				}
				columns = append(columns, cols...)
			}
		}
	case *pg.Node_AExpr:
		cols, err := parseAexpr(node)
		if err != nil {
			return nil, fmt.Errorf("resTarget expr: %w", err)
		}

		columns = append(columns, cols...)
	}

	return columns, nil
}
