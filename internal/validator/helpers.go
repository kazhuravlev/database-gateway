package validator

import (
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
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
