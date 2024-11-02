package validator

import (
	"errors"
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/just"
	"reflect"
	"strings"
)

var ErrBadQuery = errors.New("bad query")
var ErrComplicatedQuery = errors.New("complicated query")
var ErrBadConfiguration = errors.New("bad configuration")
var ErrAccessDenied = errors.New("access denied")

func IsAllowed(target config.Target, user config.User, query string) error {
	acls := just.SliceFilter(user.Acls, func(acl config.ACL) bool {
		return acl.Target == target.Id
	})
	if len(acls) == 0 {
		return fmt.Errorf("user have no any acls: %w", ErrAccessDenied)
	}

	stmts, err := parser.Parse(query)
	if err != nil {
		return fmt.Errorf("parse query: %w", err)
	}

	switch len(stmts) {
	default:
		return fmt.Errorf("query contains more than one statement: %w", ErrBadQuery)
	case 0:
		return fmt.Errorf("query contains no statements: %w", ErrBadQuery)
	case 1:
	}

	stmt := stmts[0]

	if tree.CanModifySchema(stmt.AST) {
		return fmt.Errorf("unable to modify schema: %w", ErrBadQuery)
	}

	var crudRequests []tree.NodeFormatter
	collect := func(n tree.NodeFormatter) {
		switch n := n.(type) {
		case *tree.Insert, *tree.Select, *tree.Update, *tree.Delete:
			crudRequests = append(crudRequests, n)
		}
	}
	if err := Walk3(collect, stmt.AST); err != nil {
		return fmt.Errorf("walk statement ast: %w", err)
	}

	if len(crudRequests) == 0 {
		return fmt.Errorf("unsupported query: %w", ErrBadQuery)
	}

	vectors := make([]IVector, 0, len(crudRequests))
	for _, req := range crudRequests {
		switch req := req.(type) {
		default:
			return fmt.Errorf("unexpected query: %w", ErrBadQuery)
		case *tree.Insert:
			vec, err := makeInsertVec(req)
			if err != nil {
				return fmt.Errorf("make insert vec: %w", err)
			}
			vectors = append(vectors, vec)
		case *tree.Select:
			vec, err := makeSelectVec(req)
			if err != nil {
				return fmt.Errorf("make select vec: %w", err)
			}
			vectors = append(vectors, vec)
		case *tree.Update:
			//vec, err := makeUpdateVec(req)
			//if err != nil {
			//	return fmt.Errorf("make update vec: %w", err)
			//}
			//vectors = append(vectors, vec)
		case *tree.Delete:
			//vec, err := makeDeleteVec(req)
			//if err != nil {
			//	return fmt.Errorf("make delete vec: %w", err)
			//}
			//vectors = append(vectors, vec)
		}
	}

	// Find acl for each vector.
	for _, vec := range vectors {
		isAllowed := false
		for _, acl := range acls {
			if acl.Op != vec.Op() {
				continue
			}

			if acl.Tbl != vec.Table() {
				continue
			}

			isAllowed = acl.Allow
			break
		}

		if !isAllowed {
			return fmt.Errorf("denied operation (%s): %w", vec.String(), ErrAccessDenied)
		}
	}

	return nil
}

func getTableName(t tree.TableExpr) (string, error) {
	switch t := t.(type) {
	default:
		return "", fmt.Errorf("query have complicated table name definition (%T): %w", t, ErrComplicatedQuery)
	case *tree.TableName:
		return t.FQString(), nil
	case *tree.AliasedTableExpr:
		return t.String(), nil
	}
}

type IVector interface {
	Op() config.Op
	String() string
	Table() string
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

type vecSelect struct {
	tblName string
	columns []string
}

func (vecSelect) Op() config.Op {
	return config.OpSelect
}

func (v vecSelect) String() string {
	return "select: " + v.tblName + " " + strings.Join(v.columns, ", ")
}

func (v vecSelect) Table() string {
	return v.tblName
}

func makeInsertVec(req *tree.Insert) (*vecInsert, error) {
	tName, err := getTableName(req.Table)
	if err != nil {
		return nil, fmt.Errorf("get table name for insert: %w", err)
	}

	return &vecInsert{tblName: tName}, nil
}

func makeSelectVec(req *tree.Select) (*vecSelect, error) {
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

	cols := make([]string, 0, len(sel.Exprs))
	for _, col := range sel.Exprs {
		name := col.Expr.String()
		if name == "*" {
			return nil, fmt.Errorf("unable to parse star notation: %w", ErrBadQuery)
		}

		cols = append(cols, name)
	}

	return &vecSelect{tblName: tName, columns: cols}, nil
}

//type WalkCtx struct {
//	Filter  func(tree.NodeFormatter) bool
//	Handler func(tree.NodeFormatter) error
//}

func Walk3(collect func(formatter tree.NodeFormatter), statements ...tree.NodeFormatter) error {
	for _, stmt := range statements {
		if stmt == nil {
			continue
		}

		if val := reflect.ValueOf(stmt); val.IsZero() {
			continue
		}

		collect(stmt)

		var next []tree.NodeFormatter

		switch node := stmt.(type) {
		case *tree.With:
			if node == nil {
				continue
			}

			next = append(next, just.SliceMap(node.CTEList, func(cte *tree.CTE) tree.NodeFormatter {
				return cte.Stmt
			})...)
		case *tree.Select:
			next = append(next, node.With, node.Select, &node.OrderBy, node.Limit, &node.Locking)
		case *tree.AliasedTableExpr:
			next = append(next, node.Expr, node.IndexFlags)
		case *tree.AndExpr:
			next = append(next, node.Left, node.Right)
		case *tree.AnnotateTypeExpr:
			next = append(next, node.Expr)
		case *tree.Array:
			next = append(next, just.SliceMap(node.Exprs, func(expr tree.Expr) tree.NodeFormatter {
				return expr
			})...)
		case *tree.AsOfClause:
			next = append(next, node.Expr)
		case *tree.BinaryExpr:
			next = append(next, node.Left, node.Right)
		case *tree.CaseExpr:
			next = append(next, node.Expr)
			next = append(next, just.SliceMap(node.Whens, func(w *tree.When) tree.NodeFormatter {
				return w
			})...)
			next = append(next, node.Else)
		case *tree.RangeCond:
			next = append(next, node.Left, node.From, node.To)
		case *tree.CastExpr:
			next = append(next, node.Expr)
		case *tree.CoalesceExpr:
			next = append(next, just.SliceMap(node.Exprs, func(e tree.Expr) tree.NodeFormatter {
				return e
			})...)
		case *tree.ComparisonExpr:
			next = append(next, node.Left, node.Right)
		case *tree.CreateTable:
			next = append(next, just.SliceMap(node.Defs, func(t tree.TableDef) tree.NodeFormatter {
				return t
			})...)
			next = append(next, node.AsSource)
			next = append(next, &node.StorageParams)
			next = append(next, node.PartitionBy)
			next = append(next, node.Interleave)
			next = append(next, &node.Table)
		case *tree.Exprs:
			next = append(next, just.SliceMap(*node, func(n tree.Expr) tree.NodeFormatter {
				return n
			})...)
		case *tree.From:
			next = append(next, &node.AsOf)
			next = append(next, just.SliceMap(node.Tables, func(n tree.TableExpr) tree.NodeFormatter {
				return n
			})...)
		case *tree.FuncExpr:
			next = append(next, node.WindowDef, node.Filter, &node.Exprs)
		case *tree.JoinTableExpr:
			next = append(next, node.Left, node.Right, node.Cond)
		case *tree.NotExpr:
			next = append(next, node.Expr)
		case *tree.OnJoinCond:
			next = append(next, node.Expr)
		case *tree.Order:
			next = append(next, node.Expr, &node.Table)
		case *tree.OrderBy:
			next = append(next, just.SliceMap(*node, func(n *tree.Order) tree.NodeFormatter {
				return n
			})...)
		case *tree.OrExpr:
			next = append(next, node.Left, node.Right)
		case *tree.ParenExpr:
			next = append(next, node.Expr)
		case *tree.ParenSelect:
			next = append(next, node.Select)
		case *tree.RowsFromExpr:
			next = append(next, just.SliceMap(node.Items, func(n tree.Expr) tree.NodeFormatter {
				return n
			})...)
		case *tree.Limit:
			next = append(next, node.Offset, node.Count)
		case *tree.SelectClause:
			next = append(next, &node.Exprs, node.Where, node.Having, &node.DistinctOn, &node.GroupBy, &node.From)
		case *tree.SelectExpr:
			next = append(next, node.Expr)
		case *tree.SelectExprs:
			next = append(next, just.SliceMap(*node, func(n tree.SelectExpr) tree.NodeFormatter {
				return &n
			})...)
		case *tree.SetVar:
			next = append(next, &node.Values)
		case *tree.Subquery:
			next = append(next, node.Select)
		case *tree.TableExprs:
			next = append(next, just.SliceMap(*node, func(n tree.TableExpr) tree.NodeFormatter {
				return n
			})...)
		case *tree.Tuple:
			next = append(next, &node.Exprs)
		case *tree.UnaryExpr:
			next = append(next, node.Expr)
		case *tree.UnionClause:
			next = append(next, node.Left, node.Right)
		case *tree.ValuesClause:
			next = append(next, just.SliceMap(node.Rows, func(n tree.Exprs) tree.NodeFormatter {
				return &n
			})...)
		case *tree.Where:
			next = append(next, node.Expr)
		case *tree.Window:
			next = append(next, just.SliceMap(*node, func(n *tree.WindowDef) tree.NodeFormatter {
				return n
			})...)
		case *tree.WindowDef:
			next = append(next, &node.Partitions, node.Frame)
		case *tree.WindowFrame:
			next = append(next, node.Bounds.StartBound, node.Bounds.EndBound, node.Exclusion)
		case *tree.WindowFrameBound:
			next = append(next, node.OffsetExpr)
		case *tree.Update:
			next = append(next,
				node.With,
				node.Table,
				&node.Exprs,
				&node.From,
				node.Where,
				&node.OrderBy,
				node.Limit,
				node.Returning,
			)
		case *tree.DistinctOn:
			next = append(next, just.SliceMap(*node, func(t tree.Expr) tree.NodeFormatter {
				return t
			})...)
		case *tree.GroupBy:
			next = append(next, just.Pointer(tree.Exprs(*node)))
		case *tree.LockingClause:
			next = append(next, just.SliceMap(*node, func(t *tree.LockingItem) tree.NodeFormatter {
				return t
			})...)
		case *tree.ColumnTableDef:
		case *tree.DBool:
		case *tree.FamilyTableDef:
		case *tree.IndexTableDef:
		case *tree.NumVal:
		case *tree.StrVal:
		case *tree.TableName:
		case *tree.UniqueConstraintTableDef:
		case *tree.UnqualifiedStar:
		case *tree.UnresolvedName:
		default:
			return fmt.Errorf("unknown node (%T): %w", stmt, ErrBadQuery)
		}

		if err := Walk3(collect, next...); err != nil {
			return err
		}
	}

	return nil
}
