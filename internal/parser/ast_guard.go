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
	"strings"
	"unicode/utf8"

	pg "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	MaxQueryBytes = 64 << 10
	maxTokens     = 4096
	maxSQLNesting = 64
	maxASTDepth   = 128
	maxASTNodes   = 16384
)

// Every populated field must be explicitly reviewed. Child messages are always
// traversed, including fields a statement-specific extractor does not use.
// The descriptor contract test forces review when the upstream AST changes.
var astFields = map[protoreflect.Name]string{ //nolint:gochecknoglobals
	"ParseResult":      "version stmts",
	"RawStmt":          "stmt stmt_location stmt_len",
	"SelectStmt":       "target_list from_clause where_clause group_clause values_lists sort_clause limit_offset limit_count limit_option op",
	"InsertStmt":       "relation cols select_stmt on_conflict_clause returning_list override",
	"UpdateStmt":       "relation target_list where_clause returning_list",
	"DeleteStmt":       "relation where_clause returning_list",
	"RangeVar":         "schemaname relname inh relpersistence alias location",
	"Alias":            "aliasname",
	"JoinExpr":         "jointype larg rarg quals",
	"ResTarget":        "name val location",
	"ColumnRef":        "fields location",
	"FuncCall":         "funcname args agg_star agg_distinct funcformat location",
	"A_Expr":           "kind name lexpr rexpr location",
	"BoolExpr":         "boolop args location",
	"NullTest":         "arg nulltesttype location",
	"SortBy":           "node sortby_dir sortby_nulls location",
	"OnConflictClause": "action infer target_list location",
	"InferClause":      "index_elems location",
	"IndexElem":        "name ordering nulls_ordering",
	"List":             "items",
	"A_Const":          "ival fval boolval sval bsval isnull location",
	"SetToDefault":     "type_mod location",
	"String":           "sval",
	"Integer":          "ival",
	"Float":            "fval",
	"Boolean":          "boolval",
	"BitString":        "bsval",
}

func scanInput(query string) error {
	if len(query) > MaxQueryBytes || !utf8.ValidString(query) || strings.IndexByte(query, 0) >= 0 {
		return fmt.Errorf("SQL must be valid UTF-8 without NUL and at most %d bytes: %w", MaxQueryBytes, ErrNotImplemented)
	}
	scan, err := pg.Scan(query)
	if err != nil {
		return fmt.Errorf("scan SQL: %w", err)
	}
	if len(scan.GetTokens()) > maxTokens {
		return fmt.Errorf("SQL token budget exceeded: %w", ErrNotImplemented)
	}
	depth := 0
	for _, token := range scan.GetTokens() {
		switch query[token.GetStart():token.GetEnd()] {
		case "(", "[":
			depth++
			if depth > maxSQLNesting {
				return fmt.Errorf("SQL nesting budget exceeded: %w", ErrNotImplemented)
			}
		case ")", "]":
			depth--
		}
	}

	return nil
}

func validateAST(root *pg.ParseResult) error {
	nodes := 0

	return walkAST(root.ProtoReflect(), 0, &nodes, validateNodeFields)
}

func validateNodeFields(message protoreflect.Message) error {
	name := message.Descriptor().Name()
	if name == "Node" {
		if message.WhichOneof(message.Descriptor().Oneofs().Get(0)) == nil {
			return fmt.Errorf("empty AST node: %w", ErrNotImplemented)
		}

		return nil
	}
	allowed, ok := astFields[name]
	if !ok {
		return fmt.Errorf("unsupported AST node %s: %w", name, ErrNotImplemented)
	}
	var err error
	message.Range(func(f protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if !strings.Contains(" "+allowed+" ", " "+string(f.Name())+" ") {
			err = fmt.Errorf("unsupported AST field %s.%s: %w", name, f.Name(), ErrNotImplemented)
		}

		return err == nil
	})

	return err
}

// walkAST is deliberately independent of the SQL feature handlers. It never
// skips a populated child, accepts unknown wire data, or follows unbounded trees.
//
//nolint:cyclop // All protobuf shapes are inspected in one bounded traversal.
func walkAST(message protoreflect.Message, depth int, nodes *int, visit func(protoreflect.Message) error) error {
	*nodes++
	if !message.IsValid() || depth > maxASTDepth || *nodes > maxASTNodes {
		return fmt.Errorf("invalid AST or AST budget exceeded: %w", ErrNotImplemented)
	}
	if len(message.GetUnknown()) != 0 {
		return fmt.Errorf("unknown AST wire fields: %w", ErrNotImplemented)
	}
	if err := visit(message); err != nil {
		return err
	}
	var err error
	message.Range(func(f protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if f.Kind() == protoreflect.EnumKind && f.Enum().Values().ByNumber(value.Enum()) == nil {
			err = fmt.Errorf("unknown AST enum %s: %w", f.FullName(), ErrNotImplemented)
		} else if f.Kind() == protoreflect.MessageKind {
			if f.IsList() {
				for i := 0; i < value.List().Len() && err == nil; i++ {
					err = walkAST(value.List().Get(i).Message(), depth+1, nodes, visit)
				}
			} else {
				err = walkAST(value.Message(), depth+1, nodes, visit)
			}
		}

		return err == nil
	})

	return err
}

// Dots cannot be represented unambiguously by the schema.table policy format.
// Reject them within identifier components rather than conflating quoted names.
func validIdentifier(name string) bool {
	return name != "" && len(name) <= 63 && utf8.ValidString(name) && !strings.ContainsAny(name, ".\x00")
}
