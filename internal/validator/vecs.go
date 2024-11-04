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
	"errors"
	"fmt"
	"strings"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/kazhuravlev/database-gateway/internal/config"
)

type Vec struct {
	Op   config.Op
	Tbl  string
	Cols []string
}

func (v Vec) String() string {
	return fmt.Sprintf("%s:%s(%s)", v.Op, v.Tbl, strings.Join(v.Cols, ","))
}

func makeVectors(query string) ([]Vec, error) { //nolint:cyclop
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

	var vectors []Vec
	var errs []error
	collect := func(req tree.NodeFormatter) {
		switch req := req.(type) {
		case *tree.Insert:
			vec, err := makeInsertVec(req)
			if err != nil {
				errs = append(errs, fmt.Errorf("make insert vec: %w", err))
			} else {
				vectors = append(vectors, *vec)
			}
		case *tree.Select:
			vec, err := MakeSelectVec(req)
			if err != nil {
				errs = append(errs, fmt.Errorf("make select vec: %w", err))
			} else {
				vectors = append(vectors, *vec)
			}
		case *tree.Update:
			vec, err := makeUpdateVec(req)
			if err != nil {
				errs = append(errs, fmt.Errorf("make update vec: %w", err))
			} else {
				vectors = append(vectors, *vec)
			}
		case *tree.Delete:
			vec, err := makeDeleteVec(req)
			if err != nil {
				errs = append(errs, fmt.Errorf("make delete vec: %w", err))
			} else {
				vectors = append(vectors, *vec)
			}
		}
	}
	if err := Walk(collect, stmt.AST); err != nil {
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

func makeInsertVec(req *tree.Insert) (*Vec, error) {
	tName, err := getTableName(req.Table)
	if err != nil {
		return nil, fmt.Errorf("get table name for insert: %w", err)
	}

	cols, err := GetColumnNames(req)
	if err != nil {
		return nil, fmt.Errorf("get column names: %w", err)
	}

	return &Vec{Op: config.OpInsert, Tbl: tName, Cols: cols}, nil
}

func MakeSelectVec(req *tree.Select) (*Vec, error) {
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

	return &Vec{Op: config.OpSelect, Tbl: tName, Cols: cols}, nil
}

func makeUpdateVec(req *tree.Update) (*Vec, error) {
	tName, err := getTableName(req.Table)
	if err != nil {
		return nil, fmt.Errorf("get table name for update: %w", err)
	}

	cols, err := GetColumnNames(req)
	if err != nil {
		return nil, fmt.Errorf("get column names: %w", err)
	}

	return &Vec{
		Op:   config.OpUpdate,
		Tbl:  tName,
		Cols: cols,
	}, nil
}

func makeDeleteVec(req *tree.Delete) (*Vec, error) {
	tName, err := getTableName(req.Table)
	if err != nil {
		return nil, fmt.Errorf("get table name for delete: %w", err)
	}

	cols, err := GetColumnNames(req)
	if err != nil {
		return nil, fmt.Errorf("get column names: %w", err)
	}

	return &Vec{
		Op:   config.OpDelete,
		Tbl:  tName,
		Cols: cols,
	}, nil
}
