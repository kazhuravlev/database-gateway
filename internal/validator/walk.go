package validator

import (
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/kazhuravlev/just"
	"reflect"
)

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
			//tree.Window
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
			next = append(next, node.Expr, &node.Table, &node.Index)
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
			next = append(next,
				&node.DistinctOn,
				&node.Exprs,
				&node.From,
				node.Where,
				&node.GroupBy,
				node.Having,
				&node.Window,
			)
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
			next = append(next, &node.Partitions, node.Frame, &node.OrderBy)
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
		case *tree.Delete:
			next = append(next,
				node.With,
				node.Table,
				node.Where,
				&node.OrderBy,
				node.Limit,
				node.Returning,
			)
		case *tree.Insert:
			next = append(next,
				node.With,
				node.Table,
				&node.Columns,
				node.Rows,
				node.Returning,
			)
			if node.OnConflict != nil {
				next = append(next,
					&node.OnConflict.Columns,
					&node.OnConflict.Exprs,
					node.OnConflict.Where,
				)
			}
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
		case *tree.ColumnItem:
			next = append(next, node.TableName)
		case *tree.UpdateExprs:
			next = append(next, just.SliceMap(*node, func(t *tree.UpdateExpr) tree.NodeFormatter {
				return t
			})...)
		case *tree.UpdateExpr:
			next = append(next, &node.Names, node.Expr)
		case *tree.NameList:
			next = append(next, just.SliceMap(*node, func(t tree.Name) tree.NodeFormatter {
				return &t
			})...)
		//case *tree.ColumnTableDef:
		//case *tree.DBool:
		//case *tree.FamilyTableDef:
		//case *tree.IndexTableDef:
		//case *tree.UniqueConstraintTableDef:
		//case *tree.UnqualifiedStar:
		case *tree.NoReturningClause:
		case *tree.Name:
		case *tree.NumVal:
		case *tree.StrVal:
		case *tree.TableName:
		case *tree.UnrestrictedName:
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