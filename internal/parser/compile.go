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
	"reflect"

	pg "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Compile binds unqualified relations to the configured schema, never to the
// server's search_path. The returned SQL and vectors describe the same AST.
func Compile(query, defaultSchema string) (string, []Vector, error) {
	if !validIdentifier(defaultSchema) {
		return "", nil, fmt.Errorf("invalid default schema: %w", ErrNotImplemented)
	}
	tree, err := parseTree(query)
	if err != nil {
		return "", nil, err
	}
	if err := bindNames(tree, defaultSchema); err != nil {
		return "", nil, err
	}
	vectors, err := checkedVectors(tree)
	if err != nil {
		return "", nil, err
	}
	sql, err := pg.Deparse(tree)
	if err != nil {
		return "", nil, fmt.Errorf("deparse checked AST: %w", err)
	}
	if err := verifyRoundtrip(tree, sql, vectors); err != nil {
		return "", nil, err
	}

	return sql, vectors, nil
}

func bindNames(tree *pg.ParseResult, defaultSchema string) error {
	nodes := 0

	return walkAST(tree.ProtoReflect(), 0, &nodes, func(m protoreflect.Message) error {
		switch node := m.Interface().(type) {
		case *pg.RangeVar:
			if node.GetSchemaname() == "" {
				node.Schemaname = defaultSchema
			}
		case *pg.FuncCall:
			if len(node.GetFuncname()) == 1 {
				node.Funcname = append([]*pg.Node{pg.MakeStrNode("pg_catalog")}, node.GetFuncname()...)
			}
		}

		return nil
	})
}

func verifyRoundtrip(tree *pg.ParseResult, sql string, vectors []Vector) error {
	// Do not trust a pretty-printer to preserve the authorization boundary.
	// Reparse the exact bytes destined for PostgreSQL and compare requirements.
	roundtripTree, err := parseTree(sql)
	if err != nil {
		return fmt.Errorf("verify compiled SQL: %w", err)
	}
	roundtrip, err := checkedVectors(roundtripTree)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(vectors, roundtrip) {
		return fmt.Errorf("compiled SQL changed dependencies: %w", ErrNotImplemented)
	}
	if err := clearSourceLocations(tree); err != nil {
		return err
	}
	if err := clearSourceLocations(roundtripTree); err != nil {
		return err
	}
	if !proto.Equal(tree, roundtripTree) {
		return fmt.Errorf("compiled SQL changed AST semantics: %w", ErrNotImplemented)
	}

	return nil
}

func clearSourceLocations(tree *pg.ParseResult) error {
	nodes := 0

	return walkAST(tree.ProtoReflect(), 0, &nodes, func(message protoreflect.Message) error {
		for _, name := range []protoreflect.Name{"location", "stmt_location", "stmt_len"} {
			if field := message.Descriptor().Fields().ByName(name); field != nil {
				message.Clear(field)
			}
		}

		return nil
	})
}

func checkedVectors(tree *pg.ParseResult) ([]Vector, error) {
	vectors, err := extractVectors(tree)
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("empty dependency set: %w", ErrNotImplemented)
	}
	if err := checkCoverage(tree, vectors); err != nil {
		return nil, err
	}

	return vectors, nil
}

// Independently account for every raw relation and column reference. A future
// feature handler cannot silently omit a subtree while still returning success.
func astReferences(tree *pg.ParseResult) (map[string]Columns, error) {
	tables := NewTables("public")
	var columns Columns
	nodes := 0
	err := walkAST(tree.ProtoReflect(), 0, &nodes, func(m protoreflect.Message) error {
		switch node := m.Interface().(type) {
		case *pg.RangeVar:
			return addRangeVar(tables, node)
		case *pg.ColumnRef:
			column, err := pNodeColumnRef(&pg.Node_ColumnRef{ColumnRef: node})
			if err != nil {
				return err
			}
			columns = append(columns, column)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := tables.Finalize(); err != nil {
		return nil, err
	}
	all, err := tables.GetAll()
	if err != nil {
		return nil, err
	}

	return columnsByTable(tables, all, columns)
}

//nolint:cyclop // Both missing and extraneous relation dependencies must be rejected.
func checkCoverage(tree *pg.ParseResult, vectors []Vector) error {
	refs, err := astReferences(tree)
	if err != nil {
		return err
	}
	covered := make(map[string]map[string]bool)
	for _, vector := range vectors {
		table, cols, err := vectorColumns(vector)
		if err != nil {
			return err
		}
		if _, exists := refs[table]; !exists {
			return fmt.Errorf("vector references absent relation: %w", ErrNotImplemented)
		}
		if covered[table] == nil {
			covered[table] = make(map[string]bool)
		}
		for _, col := range cols {
			covered[table][col] = true
		}
	}
	for table, cols := range refs {
		if covered[table] == nil {
			return fmt.Errorf("unaccounted relation %q: %w", table, ErrNotImplemented)
		}
		for _, col := range cols {
			if !covered[table][col.column] {
				return fmt.Errorf("unaccounted column %q: %w", col.column, ErrNotImplemented)
			}
		}
	}

	return nil
}

//nolint:gocritic // Returns the relation identity and its referenced columns.
func vectorColumns(vector Vector) (string, []string, error) {
	switch vector := vector.(type) {
	case SelectVec:
		return vector.Tbl, vector.Columns(), nil
	case InsertVec:
		return vector.Tbl, vector.Columns(), nil
	case UpdateVec:
		return vector.Tbl, vector.Columns(), nil
	case DeleteVec:
		return vector.Tbl, vector.Columns(), nil
	default:
		return "", nil, fmt.Errorf("unknown dependency vector: %w", ErrNotImplemented)
	}
}
