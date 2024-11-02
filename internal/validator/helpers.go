package validator

import (
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/kazhuravlev/just"
	"sort"
)

func getTableName(t tree.TableExpr) (string, error) {
	switch t := t.(type) {
	default:
		return "", fmt.Errorf("query have complicated table name definition (%T): %w", t, ErrComplicatedQuery)
	case *tree.TableName:
		return t.SchemaName.String() + "." + t.TableName.String(), nil
	case *tree.AliasedTableExpr:
		return t.String(), nil
	}
}

func FilterType[T tree.NodeFormatter](req tree.NodeFormatter) ([]T, error) {
	var res []T
	err := Walk3(func(node tree.NodeFormatter) {
		switch n := node.(type) {
		case T:
			res = append(res, n)
		}
	}, req)
	if err := err; err != nil {
		return nil, fmt.Errorf("filter statement: %w", err)
	}

	return res, nil
}

func GetColumnNames(req tree.NodeFormatter) ([]string, error) {
	colItems, err := FilterType[*tree.UnresolvedName](req)
	if err != nil {
		return nil, fmt.Errorf("filter columns: %w", err)
	}

	cols := just.SliceMap(colItems, func(col *tree.UnresolvedName) string {
		return col.String()
	})

	for _, col := range cols {
		if col == "*" {
			return nil, fmt.Errorf("unable to parse star notation: %w", ErrBadQuery)
		}
	}

	sort.Strings(cols)

	return cols, nil
}
