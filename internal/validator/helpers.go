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

// FilterType will filter objects with specified type.
func FilterType[T tree.NodeFormatter](req tree.NodeFormatter) ([]T, error) {
	var res []T
	err := Walk(func(node tree.NodeFormatter) {
		if n, ok := node.(T); ok {
			res = append(res, n)
		}
	}, req)
	if err := err; err != nil {
		return nil, fmt.Errorf("filter statement: %w", err)
	}

	return res, nil
}

// GetColumnNames will return all mentioned columns from query.
// Note: It will have unexpected behavior for queries that have a subquery.
func GetColumnNames(req tree.NodeFormatter) ([]string, error) {
	colItems, err := FilterType[*tree.UnresolvedName](req)
	if err != nil {
		return nil, fmt.Errorf("filter columns: %w", err)
	}

	cols := just.SliceMap(colItems, func(col *tree.UnresolvedName) string {
		return col.String()
	})

	if len(cols) == 0 {
		// FIXME: actually this is not about empty list. This is about Star notation.
		return nil, fmt.Errorf("empty column list: %w", ErrAccessDenied)
	}

	sort.Strings(cols)

	return cols, nil
}
