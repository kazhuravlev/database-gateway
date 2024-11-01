package validator

import (
	"errors"
	"fmt"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/kazhuravlev/database-gateway/internal/config"
)

var ErrBadQuery = errors.New("bad query")

func IsAllowed(target config.Target, user config.User, query string) error {
	fmt.Println(query)
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

	fmt.Println("=======================")
	fmt.Println(len(stmts), stmts)
	fmt.Println("=======================")

	//var statements []tree.Statement
	//w := &walk.AstWalker{
	//	Fn: func(ctx, node any) bool {
	//		switch n := node.(type) {
	//		case *tree.Select:
	//			statements = append(statements, tree.Statement(n))
	//		case *tree.Delete:
	//			statements = append(statements, tree.Statement(n))
	//		case *tree.Insert:
	//			statements = append(statements, tree.Statement(n))
	//		case *tree.Update:
	//			statements = append(statements, tree.Statement(n))
	//		}
	//
	//		return false
	//	},
	//}
	//
	//fmt.Println(w.Walk(stmts, nil))
	//
	//fmt.Println(statements)
	//
	//if len(statements) == 0 {
	//	return false
	//}
	//
	//return false

	return nil
}

//func Walk(stmts parser.Statements, fn func(parser.Statement) error) error {
//	for _, stmt := range stmts {
//		if err := fn(stmt); err != nil {
//			return err
//		}
//	}
//
//
//
//	var walk func(...interface{})
//	walk = func(nodes ...interface{}) {
//		for _, node := range nodes {
//			if _, ok := node.(tree.Datum); ok {
//				continue
//			}
//
//			switch node := node.(type) {
//			case *tree.AliasedTableExpr:
//				walk(node.Expr)
//			case *tree.AndExpr:
//				walk(node.Left, node.Right)
//			case *tree.AnnotateTypeExpr:
//				walk(node.Expr)
//			case *tree.Array:
//				walk(node.Exprs)
//			case tree.AsOfClause:
//				walk(node.Expr)
//			case *tree.BinaryExpr:
//				walk(node.Left, node.Right)
//			case *tree.CaseExpr:
//				walk(node.Expr, node.Else)
//				for _, when := range node.Whens {
//					walk(when.Cond, when.Val)
//				}
//			case *tree.RangeCond:
//				walk(node.Left, node.From, node.To)
//			case *tree.CastExpr:
//				walk(node.Expr)
//			case *tree.CoalesceExpr:
//				for _, expr := range node.Exprs {
//					walk(expr)
//				}
//			case *tree.ColumnTableDef:
//			case *tree.ComparisonExpr:
//				walk(node.Left, node.Right)
//			case *tree.CreateTable:
//				for _, def := range node.Defs {
//					walk(def)
//				}
//				if node.AsSource != nil {
//					walk(node.AsSource)
//				}
//			case *tree.CTE:
//				walk(node.Stmt)
//			case *tree.DBool:
//			case tree.Exprs:
//				for _, expr := range node {
//					walk(expr)
//				}
//			case *tree.FamilyTableDef:
//			case *tree.From:
//				walk(node.AsOf)
//				for _, table := range node.Tables {
//					walk(table)
//				}
//			case *tree.FuncExpr:
//				if node.WindowDef != nil {
//					walk(node.WindowDef)
//				}
//				walk(node.Exprs, node.Filter)
//			case *tree.IndexTableDef:
//			case *tree.JoinTableExpr:
//				walk(node.Left, node.Right, node.Cond)
//			case *tree.NotExpr:
//				walk(node.Expr)
//			case *tree.NumVal:
//			case *tree.OnJoinCond:
//				walk(node.Expr)
//			case *tree.Order:
//				walk(node.Expr, node.Table)
//			case tree.OrderBy:
//				for _, order := range node {
//					walk(order)
//				}
//			case *tree.OrExpr:
//				walk(node.Left, node.Right)
//			case *tree.ParenExpr:
//				walk(node.Expr)
//			case *tree.ParenSelect:
//				walk(node.Select)
//			case *tree.RowsFromExpr:
//				for _, expr := range node.Items {
//					walk(expr)
//				}
//			case *tree.Select:
//				if node.With != nil {
//					walk(node.With)
//				}
//				if node.OrderBy != nil {
//					walk(node.OrderBy)
//				}
//				if node.Limit != nil {
//					walk(node.Limit)
//				}
//				walk(node.Select)
//			case *tree.Limit:
//				walk(node.Count)
//			case *tree.SelectClause:
//				walk(node.Exprs)
//				if node.Where != nil {
//					walk(node.Where)
//				}
//				if node.Having != nil {
//					walk(node.Having)
//				}
//				if node.DistinctOn != nil {
//					for _, distinct := range node.DistinctOn {
//						walk(distinct)
//					}
//				}
//				if node.GroupBy != nil {
//					for _, group := range node.GroupBy {
//						walk(group)
//					}
//				}
//				walk(&node.From)
//			case tree.SelectExpr:
//				walk(node.Expr)
//			case tree.SelectExprs:
//				for _, expr := range node {
//					walk(expr)
//				}
//			case *tree.SetVar:
//				for _, expr := range node.Values {
//					walk(expr)
//				}
//			case *tree.StrVal:
//			case *tree.Subquery:
//				walk(node.Select)
//			case tree.TableExprs:
//				for _, expr := range node {
//					walk(expr)
//				}
//			case *tree.TableName, tree.TableName:
//			case *tree.Tuple:
//				for _, expr := range node.Exprs {
//					walk(expr)
//				}
//			case *tree.UnaryExpr:
//				walk(node.Expr)
//			case *tree.UniqueConstraintTableDef:
//			case *tree.UnionClause:
//				walk(node.Left, node.Right)
//			case tree.UnqualifiedStar:
//			case *tree.UnresolvedName:
//			case *tree.ValuesClause:
//				for _, row := range node.Rows {
//					walk(row)
//				}
//			case *tree.Where:
//				walk(node.Expr)
//			case tree.Window:
//				for _, windowDef := range node {
//					walk(windowDef)
//				}
//			case *tree.WindowDef:
//				walk(node.Partitions)
//				if node.Frame != nil {
//					walk(node.Frame)
//				}
//			case *tree.WindowFrame:
//				if node.Bounds.StartBound != nil {
//					walk(node.Bounds.StartBound)
//				}
//				if node.Bounds.EndBound != nil {
//					walk(node.Bounds.EndBound)
//				}
//			case *tree.WindowFrameBound:
//				walk(node.OffsetExpr)
//			case *tree.With:
//				for _, expr := range node.CTEList {
//					walk(expr)
//				}
//			case *tree.Update:
//				walk(node.Table)
//				for _, expr := range node.Exprs {
//					walk(expr)
//				}
//				for _, expr := range node.From {
//					walk(expr)
//				}
//				for _, expr := range node.OrderBy {
//					walk(expr)
//				}
//				if node.With != nil {
//					walk(node.With)
//				}
//				if node.Limit != nil {
//					walk(node.Limit)
//				}
//				if node.Where != nil {
//					walk(node.Where)
//				}
//				if node.Returning != nil {
//					walk(node.Returning)
//				}
//			default:
//				if w.UnknownNodes != nil {
//					w.UnknownNodes = append(w.UnknownNodes, node)
//				}
//			}
//		}
//	}
//
//
//	return nil
//}
