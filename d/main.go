package main

import (
	"fmt"

	"github.com/xwb1989/sqlparser"
)

func isAllowedQuery(query string) bool {
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		fmt.Println("Error parsing SQL:", err)
		return false
	}

	switch stmt.(type) {
	case *sqlparser.Select:
		// Allow SELECT queries only
		// Add more checks for specific columns, tables, etc.
		return true
	default:
		// Reject other query types (like INSERT, UPDATE, DELETE, etc.)
		return false
	}
}

func main() {
	query := `WITH regional_sales AS (
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
GROUP BY region, product;`
	if isAllowedQuery(query) {
		fmt.Println("Query is allowed")
	} else {
		fmt.Println("Query is not allowed")
	}
}
