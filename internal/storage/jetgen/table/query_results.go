//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package table

import (
	"github.com/go-jet/jet/v2/postgres"
)

var QueryResults = newQueryResultsTable("public", "query_results", "")

type queryResultsTable struct {
	postgres.Table

	// Columns
	ID        postgres.ColumnString
	UserID    postgres.ColumnString
	CreatedAt postgres.ColumnTimestampz
	Query     postgres.ColumnString
	Response  postgres.ColumnString

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
}

type QueryResultsTable struct {
	queryResultsTable

	EXCLUDED queryResultsTable
}

// AS creates new QueryResultsTable with assigned alias
func (a QueryResultsTable) AS(alias string) *QueryResultsTable {
	return newQueryResultsTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new QueryResultsTable with assigned schema name
func (a QueryResultsTable) FromSchema(schemaName string) *QueryResultsTable {
	return newQueryResultsTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new QueryResultsTable with assigned table prefix
func (a QueryResultsTable) WithPrefix(prefix string) *QueryResultsTable {
	return newQueryResultsTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new QueryResultsTable with assigned table suffix
func (a QueryResultsTable) WithSuffix(suffix string) *QueryResultsTable {
	return newQueryResultsTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newQueryResultsTable(schemaName, tableName, alias string) *QueryResultsTable {
	return &QueryResultsTable{
		queryResultsTable: newQueryResultsTableImpl(schemaName, tableName, alias),
		EXCLUDED:          newQueryResultsTableImpl("", "excluded", ""),
	}
}

func newQueryResultsTableImpl(schemaName, tableName, alias string) queryResultsTable {
	var (
		IDColumn        = postgres.StringColumn("id")
		UserIDColumn    = postgres.StringColumn("user_id")
		CreatedAtColumn = postgres.TimestampzColumn("created_at")
		QueryColumn     = postgres.StringColumn("query")
		ResponseColumn  = postgres.StringColumn("response")
		allColumns      = postgres.ColumnList{IDColumn, UserIDColumn, CreatedAtColumn, QueryColumn, ResponseColumn}
		mutableColumns  = postgres.ColumnList{UserIDColumn, CreatedAtColumn, QueryColumn, ResponseColumn}
	)

	return queryResultsTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:        IDColumn,
		UserID:    UserIDColumn,
		CreatedAt: CreatedAtColumn,
		Query:     QueryColumn,
		Response:  ResponseColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}