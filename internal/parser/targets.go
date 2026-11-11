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

	pg "github.com/pganalyze/pg_query_go/v6"
)

// A ResTarget means different things in INSERT columns, assignments and RETURNING.
// Never infer its meaning from Name being nonempty: RETURNING uses it as an alias.
func parseInsertColumns(nodes []*pg.Node) (Columns, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("INSERT requires explicit columns: %w", ErrNotImplemented)
	}
	columns := make(Columns, 0, len(nodes))
	seen := make(map[string]bool)
	for _, node := range nodes {
		target := node.GetResTarget()
		if target == nil || target.GetVal() != nil || !validIdentifier(target.GetName()) || seen[target.GetName()] {
			return nil, fmt.Errorf("invalid INSERT column: %w", ErrNotImplemented)
		}
		seen[target.GetName()] = true
		columns = append(columns, Column{table: nil, column: target.GetName()})
	}

	return columns, nil
}

//nolint:cyclop,nonamedreturns // Explicit shapes; named results distinguish all columns from read dependencies.
func parseAssignments(nodes []*pg.Node) (columns, reads Columns, err error) {
	seen := make(map[string]bool)
	for _, node := range nodes {
		target := node.GetResTarget()
		if target == nil || !validIdentifier(target.GetName()) || seen[target.GetName()] {
			return nil, nil, fmt.Errorf("invalid assignment target: %w", ErrNotImplemented)
		}
		seen[target.GetName()] = true
		columns = append(columns, Column{table: nil, column: target.GetName()})
		var refs Columns
		switch value := target.GetVal().GetNode().(type) {
		case *pg.Node_AConst, *pg.Node_SetToDefault:
		case *pg.Node_ColumnRef:
			col, err := pNodeColumnRef(value)
			if err != nil {
				return nil, nil, err
			}
			refs = append(refs, col)
		case *pg.Node_AExpr:
			var err error
			refs, err = parseAexpr(value)
			if err != nil {
				return nil, nil, err
			}
		default:
			return nil, nil, fmt.Errorf("unsupported assignment expression %T: %w", value, ErrNotImplemented)
		}
		columns = append(columns, refs...)
		reads = append(reads, refs...)
	}

	return columns, reads, nil
}

func parseReturning(nodes []*pg.Node) (Columns, error) {
	var columns Columns
	for _, node := range nodes {
		target := node.GetResTarget()
		if target == nil {
			return nil, fmt.Errorf("invalid RETURNING target: %w", ErrNotImplemented)
		}
		ref, ok := target.GetVal().GetNode().(*pg.Node_ColumnRef)
		if !ok {
			return nil, fmt.Errorf("unsupported RETURNING expression: %w", ErrNotImplemented)
		}
		col, err := pNodeColumnRef(ref)
		if err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}

	return columns, nil
}

func parseConflictColumns(nodes []*pg.Node) (Columns, error) {
	var columns Columns
	for _, node := range nodes {
		idx := node.GetIndexElem()
		if idx == nil || !validIdentifier(idx.GetName()) || idx.GetOrdering() != pg.SortByDir_SORTBY_DEFAULT ||
			idx.GetNullsOrdering() != pg.SortByNulls_SORTBY_NULLS_DEFAULT {
			return nil, fmt.Errorf("unsupported conflict index column: %w", ErrNotImplemented)
		}
		columns = append(columns, Column{table: nil, column: idx.GetName()})
	}

	return columns, nil
}
