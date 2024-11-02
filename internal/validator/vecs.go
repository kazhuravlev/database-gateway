package validator

import (
	"errors"
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"strings"
)

func makeVectors(query string) ([]IVector, error) {
	stmts, err := parser.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("parse query: %w", err)
	}

	if len(stmts) != 1 {
		return nil, fmt.Errorf("query must contains only one statement: %w", ErrBadQuery)
	}

	stmt := stmts[0]

	if tree.CanModifySchema(stmt.AST) {
		return nil, fmt.Errorf("unable to modify schema: %w", ErrBadQuery)
	}

	var vectors []IVector
	var errs []error
	collect := func(req tree.NodeFormatter) {
		switch req := req.(type) {
		case *tree.Insert:
			vec, err := makeInsertVec(req)
			if err != nil {
				errs = append(errs, fmt.Errorf("make insert vec: %w", err))
			} else {
				vectors = append(vectors, vec)
			}
		case *tree.Select:
			vec, err := MakeSelectVec(req)
			if err != nil {
				errs = append(errs, fmt.Errorf("make select vec: %w", err))
			} else {
				vectors = append(vectors, vec)
			}
		case *tree.Update:
			vec, err := makeUpdateVec(req)
			if err != nil {
				errs = append(errs, fmt.Errorf("make update vec: %w", err))
			} else {
				vectors = append(vectors, vec)
			}
		case *tree.Delete:
			vec, err := makeDeleteVec(req)
			if err != nil {
				errs = append(errs, fmt.Errorf("make delete vec: %w", err))
			} else {
				vectors = append(vectors, vec)
			}
		}
	}
	if err := Walk3(collect, stmt.AST); err != nil {
		return nil, fmt.Errorf("walk statement ast: %w", err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("parse query: %w", errors.Join(errs...))
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("unsupported query: %w", ErrBadQuery)
	}

	return vectors, nil
}

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

	cols, err := GetColumnNames(req)
	if err != nil {
		return nil, fmt.Errorf("get column names: %w", err)
	}

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
	tName, err := getTableName(req.Table)
	if err != nil {
		return nil, fmt.Errorf("get table name for update: %w", err)
	}

	cols, err := GetColumnNames(req)
	if err != nil {
		return nil, fmt.Errorf("get column names: %w", err)
	}

	return &vecUpdate{
		tblName: tName,
		columns: cols,
	}, nil
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
	tName, err := getTableName(req.Table)
	if err != nil {
		return nil, fmt.Errorf("get table name for delete: %w", err)
	}

	cols, err := GetColumnNames(req)
	if err != nil {
		return nil, fmt.Errorf("get column names: %w", err)
	}

	return &vecDelete{
		tblName: tName,
		columns: cols,
	}, nil
}
