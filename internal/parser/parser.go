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
	"bytes"
	"errors"
	"fmt"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

var ErrNotImplemented = errors.New("not implemented")

type Vector interface {
	isVector()
}

func Parse(query string) ([]Vector, error) { //nolint:cyclop // this is not so complicated
	result, err := pg.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if len(result.GetStmts()) != 1 {
		return nil, errors.New("expected 1 statement") //nolint:err113
	}

	root := result.GetStmts()[0]

	switch node := root.GetStmt().GetNode().(type) {
	default:
		return nil, fmt.Errorf("unsupported root node type (%T): %w", node, ErrNotImplemented)
	case *pg.Node_SelectStmt:
		res, err := handleSelect(node.SelectStmt)
		if err != nil {
			return nil, fmt.Errorf("parse select: %w", err)
		}

		return res, nil
	case *pg.Node_InsertStmt:
		res, err := handleInsert(node.InsertStmt)
		if err != nil {
			return nil, fmt.Errorf("parse insert: %w", err)
		}

		return res, nil
	case *pg.Node_UpdateStmt:
		res, err := handleUpdate(node.UpdateStmt)
		if err != nil {
			return nil, fmt.Errorf("parse update: %w", err)
		}

		return res, nil
	case *pg.Node_DeleteStmt:
		res, err := handleDelete(node.DeleteStmt)
		if err != nil {
			return nil, fmt.Errorf("parse delete: %w", err)
		}

		return res, nil
	}
}

type Tables struct {
	m map[string]string
}

func (t *Tables) Put(catalog, schema, relation, alias string) string {
	if relation == "" {
		panic("relation is empty")
	}

	fqtnBuf := bytes.NewBuffer(nil)
	if catalog != "" {
		fqtnBuf.WriteString(catalog)
		fqtnBuf.WriteString(".")
	}

	if schema != "" {
		fqtnBuf.WriteString(schema)
		fqtnBuf.WriteString(".")
	}

	fqtnBuf.WriteString(relation)

	fqtn := fqtnBuf.String()

	t.m[relation] = fqtn
	t.m[fqtn] = fqtn
	if schema == "" {
		// FIXME(zhuravlev): use default schema
		t.m["public."+relation] = fqtn
	}
	if alias != "" {
		t.m[alias] = fqtn
	}

	return fqtn
}

func (t *Tables) Get(name string) (string, bool) {
	res, ok := t.m[name]
	if !ok {
		return "", false
	}

	if res == name {
		return res, true
	}

	return t.Get(res)
}

// Finalize will check the collected tables and in case of only one table exists - creates a new Empty mapping.
func (t *Tables) Finalize() {
	all := t.GetAll()
	if len(all) == 1 {
		t.m[""] = all[0]
	}
}

func (t *Tables) Len() int {
	return len(t.GetAll())
}

func (t *Tables) GetAll() []string {
	final := make(map[string]struct{})
	for k := range t.m {
		res, ok := t.Get(k)
		if !ok {
			panic("this is impossible")
		}
		final[res] = struct{}{}
	}

	keys := make([]string, len(final))
	var i int
	for k := range final {
		keys[i] = k
		i++
	}

	return keys
}

type Column struct {
	table  []string
	column string
}

func (c Column) Table() string {
	return strings.Join(c.table, ".")
}

func ParseColumn(tokens ...string) (Column, error) {
	if len(tokens) == 0 || len(tokens) > 4 {
		return Column{}, errors.New("expected 0-4 tokens") //nolint:err113
	}

	return Column{
		table:  tokens[:len(tokens)-1],
		column: tokens[len(tokens)-1],
	}, nil
}

type Columns []Column

func (c Columns) ListNames() []string {
	res := make([]string, len(c))
	for i, col := range c {
		res[i] = col.column
	}

	return res
}
