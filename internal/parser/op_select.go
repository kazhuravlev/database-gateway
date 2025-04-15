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

type SelectVec struct { //nolint:recvcheck
	Tbl    string
	Target []string
	Filter []string
	Group  []string
	Sort   []string
}

func (s *SelectVec) Columns() []string {
	columns := just.SliceUniq(slices.Concat(s.Target, s.Group, s.Sort, s.Filter))

	return columns
}

func (SelectVec) isVector() {}

func pNodeColumnRef(node *pg.Node_ColumnRef) (Column, error) {
	var fqdnSlice []string
	for _, field := range node.ColumnRef.GetFields() {
		switch node := field.GetNode().(type) {
		default:
			return Column{}, fmt.Errorf("columnRef field (%T): %w", field.GetNode(), ErrNotImplemented)
		case *pg.Node_AStar:
			// TODO: rewrite star to concrete column list.
			return Column{}, fmt.Errorf("star expressions: %w", ErrNotImplemented)
		case *pg.Node_AConst:
			return Column{}, fmt.Errorf("constant target field (%T): %w", field.GetNode(), ErrNotImplemented)
		case *pg.Node_String_:
			fqdnSlice = append(fqdnSlice, node.String_.GetSval())
		}
	}

	column, err := ParseColumn(fqdnSlice...)
	if err != nil {
		return Column{}, fmt.Errorf("parse column: %w", err)
	}

	return column, nil
}

func handleSelect(sel *pg.SelectStmt) ([]Vector, error) { //nolint:gocyclo,gocognit,cyclop,funlen,maintidx
	if sel.DistinctClause != nil ||
		sel.GetIntoClause() != nil ||
		sel.GetHavingClause() != nil ||
		sel.WindowClause != nil ||
		sel.GetGroupDistinct() ||
		sel.GetOp() != pg.SetOperation_SETOP_NONE ||
		sel.LockingClause != nil ||
		sel.GetWithClause() != nil ||
		sel.ValuesLists != nil ||
		sel.GetLarg() != nil ||
		sel.GetRarg() != nil ||
		sel.GetAll() {
		return nil, fmt.Errorf("unknown clause: %w", ErrNotImplemented)
	}

	if len(sel.GetFromClause()) > 1 {
		return nil, errors.New("from clause too big") //nolint:err113
	}

	tables := Tables{m: make(map[string]string)}
	from := sel.GetFromClause()[0]
	switch fromNode := from.GetNode().(type) {
	default:
		return nil, fmt.Errorf("from type (%T): %w", from.GetNode(), ErrNotImplemented)
	case *pg.Node_JoinExpr:
		return nil, fmt.Errorf("join expression: %w", ErrNotImplemented)
	case *pg.Node_RangeVar:
		tbl := fromNode.RangeVar
		if tbl.GetAlias() != nil {
			tables.Put(tbl.GetCatalogname(), tbl.GetSchemaname(), tbl.GetRelname(), tbl.GetAlias().GetAliasname())
		} else {
			tables.Put(tbl.GetCatalogname(), tbl.GetSchemaname(), tbl.GetRelname(), "")
		}
	}

	tables.Finalize()

	var allColumns Columns
	// handle target fields
	for _, target := range sel.GetTargetList() {
		switch node := target.GetNode().(type) {
		default:
			return nil, fmt.Errorf("target type (%T): %w", target.GetNode(), ErrNotImplemented)
		case *pg.Node_ResTarget:
			resTarget := node.ResTarget
			if resTarget.Indirection != nil {
				return nil, fmt.Errorf("resTarget field: %w", ErrNotImplemented)
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
			case *pg.Node_FuncCall:
				funcCall := node.FuncCall
				if funcCall.GetOver() != nil || funcCall.GetAggFilter() != nil || funcCall.AggOrder != nil {
					return nil, fmt.Errorf("window functions (%T): %w", funcCall.GetOver(), ErrNotImplemented)
				}

				if len(funcCall.GetFuncname()) != 1 {
					return nil, fmt.Errorf("only one function call support for one target: %w", ErrNotImplemented)
				}

				switch node := funcCall.GetFuncname()[0].GetNode().(type) {
				default:
					return nil, fmt.Errorf("unknown function name type (%T): %w", node, ErrNotImplemented)
				case *pg.Node_String_:
					switch name := node.String_.GetSval(); name {
					default:
						return nil, fmt.Errorf("unknown function name (%s): %w", name, ErrNotImplemented)
					case "count", "lower", "upper":
						// NOTE: allowed function names
					}
				}

				for _, node := range funcCall.GetArgs() {
					switch node := node.GetNode().(type) {
					default:
						return nil, fmt.Errorf("unknown function argument type (%T)", node) //nolint:err113
					case *pg.Node_ColumnRef:
						column, err := pNodeColumnRef(node)
						if err != nil {
							return nil, fmt.Errorf("parse column: %w", err)
						}
						allColumns = append(allColumns, column)
					}
				}
			}
		}
	}

	if sel.GetWhereClause() != nil {
		switch node := sel.GetWhereClause().GetNode().(type) { //nolint:gocritic
		case *pg.Node_AExpr:
			switch node.AExpr.GetKind() { //nolint:exhaustive // this is white list
			default:
				return nil, fmt.Errorf("where clause kind (%d): %w", node.AExpr.GetKind(), ErrNotImplemented)
			case pg.A_Expr_Kind_AEXPR_OP:
				columns, err := parseAexpr(node)
				if err != nil {
					return nil, fmt.Errorf("parse where clause aexpr: %w", err)
				}

				allColumns = append(allColumns, columns...)
			}
		}
	}

	for _, node := range sel.GetSortClause() {
		switch node := node.GetNode().(type) {
		default:
			return nil, fmt.Errorf("sort clause (%T): %w", node, ErrNotImplemented)
		case *pg.Node_SortBy:
			if node.SortBy.UseOp != nil {
				return nil, fmt.Errorf("sort useOp: %w", ErrNotImplemented)
			}

			switch node := node.SortBy.GetNode().GetNode().(type) {
			default:
				return nil, fmt.Errorf("sort clause (%T): %w", node, ErrNotImplemented)
			case *pg.Node_ColumnRef:
				column, err := pNodeColumnRef(node)
				if err != nil {
					return nil, fmt.Errorf("parse column: %w", err)
				}

				allColumns = append(allColumns, column)
			}
		}
	}

	// NOTE: Implement checking of sel.LimitOffset
	// NOTE: Implement checking of sel.LimitCount
	// NOTE: Implement checking of sel.LimitOption

	for _, node := range sel.GetGroupClause() {
		switch node := node.GetNode().(type) {
		default:
			return nil, fmt.Errorf("group by node (%T): %w", node, ErrNotImplemented)
		case *pg.Node_ColumnRef:
			column, err := pNodeColumnRef(node)
			if err != nil {
				return nil, fmt.Errorf("parse group by column: %w", err)
			}

			allColumns = append(allColumns, column)
		}
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
		vectors = append(vectors, SelectVec{
			Tbl:    tbl,
			Target: cols.ListNames(),
			Filter: nil, // TODO: impl
			Group:  nil, // TODO: impl
			Sort:   nil, // TODO: impl
		})
	}

	return vectors, nil
}
