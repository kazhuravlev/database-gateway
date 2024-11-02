package validator

import (
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"sort"
	"strings"
)

type vecInsert struct {
	tblName string
}

func (vecInsert) Op() config.Op {
	return config.OpInsert
}

func (v vecInsert) String() string {
	return "insert: " + v.tblName
}

func (v vecInsert) Table() string {
	return v.tblName
}

func makeInsertVec(req *tree.Insert) (*vecInsert, error) {
	tName, err := getTableName(req.Table)
	if err != nil {
		return nil, fmt.Errorf("get table name for insert: %w", err)
	}

	return &vecInsert{tblName: tName}, nil
}

type VecSelect struct {
	Tbl  string
	Cols []string
}

func (VecSelect) Op() config.Op {
	return config.OpSelect
}

func (v VecSelect) String() string {
	return "select: " + v.Tbl + " " + strings.Join(v.Cols, ", ")
}

func (v VecSelect) Table() string {
	return v.Tbl
}

func MakeSelectVec(req *tree.Select) (*VecSelect, error) {
	sel, ok := req.Select.(*tree.SelectClause)
	if !ok {
		return nil, fmt.Errorf("query have complicated select definition: %w", ErrComplicatedQuery)
	}

	if len(sel.From.Tables) != 1 {
		return nil, fmt.Errorf("select have a several tables: %w", ErrComplicatedQuery)
	}

	tName, err := getTableName(sel.From.Tables[0])
	if err != nil {
		return nil, fmt.Errorf("get table name for select: %w", err)
	}

	cols := make([]string, 0)
	//for _, col := range sel.Exprs {
	//	name := col.Expr.String()
	//	if name == "*" {
	//		return nil, fmt.Errorf("unable to parse star notation: %w", ErrBadQuery)
	//	}
	//
	//	cols = append(cols, name)
	//}
	err2 := Walk3(func(node tree.NodeFormatter) {
		switch col := node.(type) {
		case *tree.ColumnItem:
			cols = append(cols, col.String())
		}
	}, req)
	if err := err2; err != nil {
		return nil, fmt.Errorf("walk seelct statement: %w", err)
	}

	for _, col := range cols {
		if col == "*" {
			return nil, fmt.Errorf("unable to parse star notation: %w", ErrBadQuery)
		}
	}

	sort.Strings(cols)

	return &VecSelect{Tbl: tName, Cols: cols}, nil
}

type vecUpdate struct {
	tblName string
	columns []string
}

func (vecUpdate) Op() config.Op {
	return config.OpUpdate
}

func (v vecUpdate) String() string {
	return "update: " + v.tblName + " " + strings.Join(v.columns, ", ")
}

func (v vecUpdate) Table() string {
	return v.tblName
}

func makeUpdateVec(req *tree.Update) (*vecUpdate, error) {
	return nil, nil
	//sel, ok := req.Select.(*tree.SelectClause)
	//if !ok {
	//	return nil, fmt.Errorf("query have complicated select definition: %w", ErrComplicatedQuery)
	//}
	//
	//if len(sel.From.Tables) != 1 {
	//	return nil, fmt.Errorf("select have a several tables: %w", ErrComplicatedQuery)
	//}
	//
	//tName, err := getTableName(sel.From.Tables[0])
	//if err != nil {
	//	return nil, fmt.Errorf("get table name for select: %w", err)
	//}
	//
	//cols := make([]string, 0, len(sel.Exprs))
	//for _, col := range sel.Exprs {
	//	name := col.Expr.String()
	//	if name == "*" {
	//		return nil, fmt.Errorf("unable to parse star notation: %w", ErrBadQuery)
	//	}
	//
	//	cols = append(cols, name)
	//}
	//
	//return &vecUpdate{Tbl: tName, Cols: cols}, nil
}

type vecDelete struct {
	tblName string
	columns []string
}

func (vecDelete) Op() config.Op {
	return config.OpDelete
}

func (v vecDelete) String() string {
	return "delete: " + v.tblName + " " + strings.Join(v.columns, ", ")
}

func (v vecDelete) Table() string {
	return v.tblName
}

func makeDeleteVec(req *tree.Delete) (*vecDelete, error) {
	//sel, ok := req.Select.(*tree.SelectClause)
	//if !ok {
	//	return nil, fmt.Errorf("query have complicated select definition: %w", ErrComplicatedQuery)
	//}
	//
	//if len(sel.From.Tables) != 1 {
	//	return nil, fmt.Errorf("select have a several tables: %w", ErrComplicatedQuery)
	//}
	//
	//tName, err := getTableName(sel.From.Tables[0])
	//if err != nil {
	//	return nil, fmt.Errorf("get table name for select: %w", err)
	//}
	//
	//cols := make([]string, 0, len(sel.Exprs))
	//for _, col := range sel.Exprs {
	//	name := col.Expr.String()
	//	if name == "*" {
	//		return nil, fmt.Errorf("unable to parse star notation: %w", ErrBadQuery)
	//	}
	//
	//	cols = append(cols, name)
	//}
	//
	//return &vecDelete{Tbl: tName, Cols: cols}, nil
	return nil, nil

}
