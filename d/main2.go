package main

import (
	"fmt"
	"log"

	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/kazhuravlev/database-gateway/walk"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
)

var validQueries = []string{
	`WITH regional_sales AS (
    SELECT region, SUM(amount) AS total_sales
    FROM orders
    GROUP BY region
), top_regions AS (
    SELECT region
    FROM regional_sales
    WHERE total_sales > (SELECT SUM(total_sales)/10 FROM regional_sales)
)
SELECT region,
       product,
       SUM(quantity) AS product_units,
       SUM(amount) AS product_sales
FROM orders
WHERE region IN (SELECT region FROM top_regions)
GROUP BY region, product;`,
}

var badQueries = []string{
	`update clients set name="adasd" where o = 11 returning id`,
}

func main() {
	sql := badQueries[0]
	// sql := validQueries[0]
	w := &walk.AstWalker{
		Fn: func(ctx, node any) (stop bool) {
			switch n := node.(type) {
			case *tree.Update:
				fmt.Println(n)
			}

			log.Printf("node type %T", node)
			return false
		},
	}

	stmts, err := parser.Parse(sql)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(w.Walk(stmts, nil))
	fmt.Println(w.UnknownNodes)
	return
}
