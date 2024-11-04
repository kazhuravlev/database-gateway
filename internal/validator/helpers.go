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

package validator

import (
	"fmt"
	"sort"

	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/kazhuravlev/just"
)

func getTableName(tbl tree.TableExpr) (string, error) {
	switch tbl := tbl.(type) {
	default:
		return "", fmt.Errorf("query have complicated table name definition (%T): %w", tbl, ErrComplicatedQuery)
	case *tree.TableName:
		if tbl.SchemaName != "" {
			return tbl.SchemaName.String() + "." + tbl.TableName.String(), nil
		}

		return "public." + tbl.TableName.String(), nil
	case *tree.AliasedTableExpr:
		return getTableName(tbl.Expr)
	}
}

// GetColumnNames will return all mentioned columns from query.
// Note: It will have unexpected behavior for queries that have a subquery.
func GetColumnNames(req tree.NodeFormatter) ([]string, error) {
	colItems := CollectType[*tree.UnresolvedName]()
	colNames := CollectType[*tree.Name]()
	colTupleStars := CollectType[*tree.TupleStar]()
	colStars := CollectType[tree.UnqualifiedStar]()

	err := Walk(CollectAll(colItems.Collect, colNames.Collect, colTupleStars.Collect, colStars.Collect), req)
	if err := err; err != nil {
		return nil, fmt.Errorf("collect types: %w", err)
	}

	var hasStar bool
	if len(colStars.Res()) != 0 || len(colTupleStars.Res()) != 0 {
		hasStar = true
	}

	cols := just.SliceMap(colItems.Res(), func(col *tree.UnresolvedName) string {
		hasStar = hasStar || col.Star
		return col.String()
	})

	cols = append(cols, just.SliceMap(colNames.Res(), func(col *tree.Name) string {
		return col.String()
	})...)

	if hasStar {
		return nil, fmt.Errorf("star expressions is restricted: %w", ErrComplicatedQuery)
	}

	if len(cols) == 0 {
		// FIXME: actually this is not about empty list. This is about Star notation.
		return nil, fmt.Errorf("empty column list: %w", ErrAccessDenied)
	}

	sort.Strings(cols)

	return cols, nil
}
