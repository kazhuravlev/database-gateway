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
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrRelationEmpty  = errors.New("relation cannot be empty")
)

type Vector interface {
	isVector()
}

func Parse(query string) ([]Vector, error) {
	result, err := parseTree(query)
	if err != nil {
		return nil, err
	}

	return checkedVectors(result)
}

func parseTree(query string) (*pg.ParseResult, error) {
	if err := scanInput(query); err != nil {
		return nil, err
	}
	result, err := pg.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if len(result.GetStmts()) != 1 {
		return nil, errors.New("expected 1 statement") //nolint:err113
	}
	if err := validateAST(result); err != nil {
		return nil, err
	}

	return result, nil
}

func extractVectors(result *pg.ParseResult) ([]Vector, error) {
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
	m             map[string]string
	relations     map[string]struct{}
	defaultSchema string
	sources       int
}

func NewTables(defaultSchema string) *Tables {
	return &Tables{
		m:             make(map[string]string),
		relations:     make(map[string]struct{}),
		defaultSchema: defaultSchema,
		sources:       0,
	}
}

//nolint:cyclop // Validate each identifier and binding before mutating the symbol table.
func (t *Tables) Put(catalog, schema, relation, alias string) (string, error) {
	if relation == "" {
		return "", ErrRelationEmpty
	}

	if catalog != "" || !validIdentifier(relation) || (schema != "" && !validIdentifier(schema)) ||
		(alias != "" && !validIdentifier(alias)) {
		return "", fmt.Errorf("unsupported relation identifier: %w", ErrNotImplemented)
	}
	fqtn := relation
	if schema != "" {
		fqtn = schema + "." + relation
	}
	keys := []string{relation, fqtn}
	if schema == "" && t.defaultSchema != "" {
		keys = append(keys, t.defaultSchema+"."+relation)
	}
	if alias != "" {
		keys = []string{alias}
	}
	// Bind names directly to immutable relation identities. An alias is never
	// followed as another alias, and can never remove a source from the ACL.
	for _, key := range keys {
		if _, exists := t.m[key]; exists {
			return "", fmt.Errorf("duplicate or ambiguous table binding %q: %w", key, ErrNotImplemented)
		}
	}
	for _, key := range keys {
		t.m[key] = fqtn
	}
	t.relations[fqtn] = struct{}{}
	t.sources++

	return fqtn, nil
}

func (t *Tables) SourcesCount() int {
	return t.sources
}

func (t *Tables) Get(name string) (string, bool) {
	res, ok := t.m[name]

	return res, ok
}

// Finalize will check the collected tables and in case of only one source table exists - creates a new Empty mapping.
func (t *Tables) Finalize() error {
	all, err := t.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get all tables: %w", err)
	}
	if t.sources == 1 && len(all) == 1 {
		t.m[""] = all[0]
	}

	return nil
}

func (t *Tables) Len() (int, error) {
	all, err := t.GetAll()
	if err != nil {
		return 0, err
	}

	return len(all), nil
}

func (t *Tables) GetAll() ([]string, error) {
	final := make(map[string]struct{})
	for relation := range t.relations {
		final[relation] = struct{}{}
	}

	keys := make([]string, len(final))
	var i int
	for k := range final {
		keys[i] = k
		i++
	}

	slices.Sort(keys)

	return keys, nil
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
		return Column{}, errors.New("expected 1-4 tokens") //nolint:err113
	}
	for _, token := range tokens {
		if !validIdentifier(token) {
			return Column{}, fmt.Errorf("unsupported column identifier: %w", ErrNotImplemented)
		}
	}

	return Column{
		table:  slices.Clone(tokens[:len(tokens)-1]),
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

func columnsByTable(tables *Tables, allTables []string, columns Columns) (map[string]Columns, error) {
	res := make(map[string]Columns, len(allTables))
	for _, tableName := range allTables {
		res[tableName] = nil
	}

	for _, column := range columns {
		if column.Table() == "" && tables.SourcesCount() > 1 {
			return nil, fmt.Errorf("ambiguous column reference (%s): %w", column.column, ErrNotImplemented)
		}

		tbl, ok := tables.Get(column.Table())
		if !ok {
			return nil, fmt.Errorf("table not found: %s", column.Table()) //nolint:err113
		}

		res[tbl] = append(res[tbl], column)
	}

	return res, nil
}
